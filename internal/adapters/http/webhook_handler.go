package http

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/dan/claude-control/internal/adapters/claude"
	"github.com/dan/claude-control/internal/core/domain"
	"github.com/dan/claude-control/internal/core/services"
	"github.com/gorilla/mux"
)

// WebhookHandler handles Claude Code webhook requests with validation
type WebhookHandler struct {
	taskService   *services.TaskService
	claudeAdapter *claude.ClaudeCodeAdapter
	maxBodySize   int64
	stopInput     string // User-configured input for Stop webhooks
}

// ClaudeCodeWebhookRequest represents the expected structure of Claude Code webhook requests
type ClaudeCodeWebhookRequest struct {
	HookEventName   string                 `json:"hook_event_name"`
	SessionID       string                 `json:"session_id"`
	CWD             string                 `json:"cwd"`
	TranscriptPath  string                 `json:"transcript_path,omitempty"`
	ToolName        string                 `json:"tool_name,omitempty"`
	ToolInput       *ToolInput             `json:"tool_input,omitempty"`
	ToolResponse    *ToolResponse          `json:"tool_response,omitempty"`
	Message         string                 `json:"message,omitempty"`
	RawData         map[string]interface{} `json:"-"` // Store raw data for flexibility
}

// ToolInput represents tool input parameters
type ToolInput struct {
	Command     string `json:"command,omitempty"`
	Description string `json:"description,omitempty"`
}

// ToolResponse represents tool execution results
type ToolResponse struct {
	Interrupted bool   `json:"interrupted"`
	IsImage     bool   `json:"isImage"`
	Stderr      string `json:"stderr"`
	Stdout      string `json:"stdout"`
}

var (
	// Valid hook event names based on observed patterns
	validHookTypes = map[string]bool{
		"PreToolUse":        true,
		"PostToolUse":       true,
		"Notification":      true,
		"UserPromptSubmit":  true,
		"Stop":              true,
		"SubagentStop":      true,
		"PreCompact":        true,
	}

	// Session ID validation (UUID format)
	sessionIDRegex = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
)

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(taskService *services.TaskService) *WebhookHandler {
	return &WebhookHandler{
		taskService:   taskService,
		claudeAdapter: claude.NewClaudeCodeAdapter(""), // Use default claude binary path
		maxBodySize:   1024 * 1024,                     // 1MB max body size
		stopInput:     "continue",                      // Default stop input
	}
}

// SetStopInput configures the input to send for Stop webhooks
func (h *WebhookHandler) SetStopInput(input string) {
	h.stopInput = input
}

// GetStopInput returns the current stop input configuration
func (h *WebhookHandler) GetStopInput() string {
	return h.stopInput
}

// RegisterRoutes registers webhook routes with the router
func (h *WebhookHandler) RegisterRoutes(router *mux.Router) {
	// Non-blocking webhook handlers (immediate response, create tasks for logging)
	router.HandleFunc("/webhook/notification", h.handleNotification).Methods("POST")
	router.HandleFunc("/webhook/stop", h.handleStop).Methods("POST")
	router.HandleFunc("/webhook/subagent-stop", h.handleSubagentStop).Methods("POST")

	// Non-blocking webhook handlers (immediate response, no task creation)
	router.HandleFunc("/webhook/pre-tool-use", h.handlePreToolUse).Methods("POST")
	router.HandleFunc("/webhook/post-tool-use", h.handlePostToolUse).Methods("POST")
	router.HandleFunc("/webhook/user-prompt-submit", h.handleUserPromptSubmit).Methods("POST")
	router.HandleFunc("/webhook/pre-compact", h.handlePreCompact).Methods("POST")

	// Generic webhook handler for any hook type
	router.HandleFunc("/webhook/{hookType}", h.handleGenericWebhook).Methods("POST")
}

// parseAndValidateRequest parses and validates the incoming webhook request
func (h *WebhookHandler) parseAndValidateRequest(r *http.Request) (*ClaudeCodeWebhookRequest, error) {
	// Validate content type
	contentType := r.Header.Get("Content-Type")
	if contentType != "" && !strings.Contains(contentType, "application/json") {
		log.Printf("⚠️  Unexpected content type: %s", contentType)
	}

	// Read the request body directly
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	if len(bodyBytes) == 0 {
		return &ClaudeCodeWebhookRequest{}, nil // Allow empty bodies
	}

	// Parse JSON into raw data first
	var rawData map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &rawData); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Parse into structured format
	var req ClaudeCodeWebhookRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to parse webhook structure: %w", err)
	}

	req.RawData = rawData

	// Validate the request
	if err := h.validateRequest(&req); err != nil {
		return nil, err
	}

	return &req, nil
}

// validateRequest performs validation on the parsed request
func (h *WebhookHandler) validateRequest(req *ClaudeCodeWebhookRequest) error {
	// Validate hook event name if present
	if req.HookEventName != "" && !validHookTypes[req.HookEventName] {
		log.Printf("⚠️  Unknown hook type: %s", req.HookEventName)
	}

	// Validate session ID format if present
	if req.SessionID != "" && !sessionIDRegex.MatchString(req.SessionID) {
		log.Printf("⚠️  Invalid session ID format: %s", req.SessionID)
	}

	// Validate paths if present
	if req.CWD != "" && !h.isValidPath(req.CWD) {
		log.Printf("⚠️  Potentially unsafe CWD path: %s", req.CWD)
	}

	if req.TranscriptPath != "" && !h.isValidPath(req.TranscriptPath) {
		log.Printf("⚠️  Potentially unsafe transcript path: %s", req.TranscriptPath)
	}

	// Validate command length if present
	if req.ToolInput != nil && req.ToolInput.Command != "" {
		if len(req.ToolInput.Command) > 5000 {
			return fmt.Errorf("command too long: %d characters (max 5000)", len(req.ToolInput.Command))
		}
		
		// Log suspicious commands
		if h.isSuspiciousCommand(req.ToolInput.Command) {
			log.Printf("⚠️  Suspicious command detected: %s", req.ToolInput.Command)
		}
	}

	return nil
}

// isValidPath performs basic path validation
func (h *WebhookHandler) isValidPath(path string) bool {
	// Allow empty paths
	if path == "" {
		return true
	}

	// Basic length check
	if len(path) > 500 {
		return false
	}

	// Check for null bytes and other dangerous characters
	if strings.ContainsAny(path, "\x00\r\n") {
		return false
	}

	return true
}

// isSuspiciousCommand checks for potentially dangerous command patterns
func (h *WebhookHandler) isSuspiciousCommand(command string) bool {
	suspiciousPatterns := []string{
		"rm -rf /",
		"format c:",
		"del /f /s /q",
		"DROP TABLE",
		"<script",
		"javascript:",
		"eval(",
		"system(",
		"exec(",
	}

	lowerCommand := strings.ToLower(command)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerCommand, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// convertToLegacyPayload converts validated request to legacy map format for domain layer
func (h *WebhookHandler) convertToLegacyPayload(req *ClaudeCodeWebhookRequest) map[string]interface{} {
	if req.RawData != nil {
		return req.RawData
	}
	
	// Fallback: construct from structured data
	payload := make(map[string]interface{})
	
	if req.HookEventName != "" {
		payload["hook_event_name"] = req.HookEventName
	}
	if req.SessionID != "" {
		payload["session_id"] = req.SessionID
	}
	if req.CWD != "" {
		payload["cwd"] = req.CWD
	}
	if req.TranscriptPath != "" {
		payload["transcript_path"] = req.TranscriptPath
	}
	if req.ToolName != "" {
		payload["tool_name"] = req.ToolName
	}
	if req.ToolInput != nil {
		payload["tool_input"] = req.ToolInput
	}
	if req.ToolResponse != nil {
		payload["tool_response"] = req.ToolResponse
	}
	if req.Message != "" {
		payload["message"] = req.Message
	}
	
	return payload
}

// handlePostToolUse handles PostToolUse webhook events
func (h *WebhookHandler) handlePostToolUse(w http.ResponseWriter, r *http.Request) {
	h.handleNonBlockingWebhook(w, r, domain.HookTypePostToolUse)
}

// handleStop handles Stop webhook events with immediate response and task creation
func (h *WebhookHandler) handleStop(w http.ResponseWriter, r *http.Request) {
	// Parse and validate the incoming webhook request
	validatedReq, err := h.parseAndValidateRequest(r)
	if err != nil {
		log.Printf("Validation failed for Stop webhook: %v", err)
		h.respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Validation error: %v", err))
		return
	}

	// Convert validated request to structured hook data
	payload := h.convertToLegacyPayload(validatedReq)
	hookData := domain.NewHookData(domain.HookTypeStop, payload)

	// Log received Claude Code data for debugging
	log.Printf("Received Stop webhook: session_id=%s, cwd=%s", 
		hookData.GetSessionID(), validatedReq.CWD)

	// Create task for logging/monitoring (non-blocking)
	if h.taskService != nil {
		task := domain.NewTask(hookData)
		if err := h.taskService.CreateTask(r.Context(), task); err != nil {
			log.Printf("Failed to create Stop task: %v", err)
			// Don't fail the webhook - this is just for logging
		} else {
			log.Printf("Created Stop task %s for session %s", task.ID.String()[:8], hookData.GetSessionID())
		}
	}

	// Return immediate non-blocking response
	hookResponse := &domain.HookResponse{
		Continue:       true,
		SuppressOutput: true, // Hide output for cleaner Claude Code behavior
	}

	log.Printf("Processed Stop webhook with immediate response: %s", hookResponse.String())
	h.respondWithJSON(w, http.StatusOK, hookResponse)
}

// handleSubagentStop handles SubagentStop webhook events with immediate response and task creation
func (h *WebhookHandler) handleSubagentStop(w http.ResponseWriter, r *http.Request) {
	h.handleNonBlockingWebhook(w, r, domain.HookTypeSubagentStop)
}

// handlePreCompact handles non-blocking PreCompact webhook events
func (h *WebhookHandler) handlePreCompact(w http.ResponseWriter, r *http.Request) {
	h.handleNonBlockingWebhook(w, r, domain.HookTypePreCompact)
}

// handleGenericWebhook handles webhook events with hook type from URL path
func (h *WebhookHandler) handleGenericWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hookTypeStr := vars["hookType"]

	hookType, err := domain.ParseHookType(hookTypeStr)
	if err != nil {
		// For unknown hook types, treat as non-blocking PreToolUse
		log.Printf("⚠️  Unknown hook type from URL: %s, treating as PreToolUse", hookTypeStr)
		hookType = domain.HookTypePreToolUse
	}

	h.handleNonBlockingWebhook(w, r, hookType)
}

// handleNonBlockingWebhook handles webhooks that don't require user approval
func (h *WebhookHandler) handleNonBlockingWebhook(w http.ResponseWriter, r *http.Request, hookType domain.HookType) {
	// Parse and validate the incoming webhook request
	validatedReq, err := h.parseAndValidateRequest(r)
	if err != nil {
		log.Printf("Validation failed for %s: %v", hookType.String(), err)
		log.Printf("Expected JSON format for %s: %s", hookType.String(), GetExpectedJSONFormat(hookType.String()))
		h.respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Validation error: %v", err))
		return
	}

	// Convert validated request to structured hook data
	payload := h.convertToLegacyPayload(validatedReq)

	// Extract hook type from Claude Code JSON (prefer hook_event_name over our URL-based hookType)
	if claudeHookType, ok := payload["hook_event_name"]; ok {
		if hookTypeStr, ok := claudeHookType.(string); ok {
			if parsedType, err := domain.ParseHookType(hookTypeStr); err == nil {
				hookType = parsedType
			}
		}
	}

	// Create structured hook data
	hookData := domain.NewHookData(hookType, payload)

	// Log received Claude Code data for debugging
	log.Printf("Received %s webhook: session_id=%s, tool_name=%s", 
		hookType.String(), hookData.GetSessionID(), hookData.GetToolName())

	// Create task for certain hook types that need logging/monitoring
	createTask := false
	switch hookType {
	case domain.HookTypePreToolUse, domain.HookTypeUserPromptSubmit:
		createTask = true // These still may need user interaction
	}

	if createTask && h.taskService != nil {
		task := domain.NewTask(hookData)
		if err := h.taskService.CreateTask(r.Context(), task); err != nil {
			log.Printf("Failed to create %s task: %v", hookType.String(), err)
			// Don't fail the webhook - this is just for logging
		} else {
			log.Printf("Created %s task %s for session %s", hookType.String(), task.ID.String()[:8], hookData.GetSessionID())
		}
	}

	// Return immediate non-blocking response
	hookResponse := &domain.HookResponse{
		Continue:       true,
		SuppressOutput: true, // Hide output for cleaner Claude Code behavior
	}

	// Log the webhook event
	log.Printf("Processed %s webhook with immediate response: %s", hookType.String(), hookResponse.String())

	// Send Claude Code compliant JSON response
	h.respondWithJSON(w, http.StatusOK, hookResponse)
}

// respondWithError sends an error response
func (h *WebhookHandler) respondWithError(w http.ResponseWriter, statusCode int, message string) {
	h.respondWithJSON(w, statusCode, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// handlePreToolUse handles non-blocking PreToolUse webhook events
func (h *WebhookHandler) handlePreToolUse(w http.ResponseWriter, r *http.Request) {
	h.handleNonBlockingWebhook(w, r, domain.HookTypePreToolUse)
}

// handleNotification handles Notification webhook events with immediate response and task creation
func (h *WebhookHandler) handleNotification(w http.ResponseWriter, r *http.Request) {
	// Parse and validate the incoming webhook request
	validatedReq, err := h.parseAndValidateRequest(r)
	if err != nil {
		log.Printf("Validation failed for Notification webhook: %v", err)
		h.respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Validation error: %v", err))
		return
	}

	// Convert validated request to structured hook data
	payload := h.convertToLegacyPayload(validatedReq)
	hookData := domain.NewHookData(domain.HookTypeNotification, payload)

	// Log received Claude Code data for debugging
	log.Printf("Received Notification webhook: session_id=%s, message=%s", 
		hookData.GetSessionID(), validatedReq.Message)

	// Create task for logging/monitoring (non-blocking)
	if h.taskService != nil {
		task := domain.NewTask(hookData)
		if err := h.taskService.CreateTask(r.Context(), task); err != nil {
			log.Printf("Failed to create Notification task: %v", err)
			// Don't fail the webhook - this is just for logging
		} else {
			log.Printf("Created Notification task %s for session %s", task.ID.String()[:8], hookData.GetSessionID())
		}
	}

	// Return immediate non-blocking response
	hookResponse := &domain.HookResponse{
		Continue:       true,
		SuppressOutput: false, // Show notification to user
	}

	log.Printf("Processed Notification webhook with immediate response: %s", hookResponse.String())
	h.respondWithJSON(w, http.StatusOK, hookResponse)
}

// handleUserPromptSubmit handles non-blocking UserPromptSubmit webhook events
func (h *WebhookHandler) handleUserPromptSubmit(w http.ResponseWriter, r *http.Request) {
	h.handleNonBlockingWebhook(w, r, domain.HookTypeUserPromptSubmit)
}

// handleBlockingWebhook is the core blocking webhook handler logic
func (h *WebhookHandler) handleBlockingWebhook(w http.ResponseWriter, r *http.Request, hookType domain.HookType) {
	// Parse and validate the incoming webhook request
	validatedReq, err := h.parseAndValidateRequest(r)
	if err != nil {
		log.Printf("Validation failed for %s: %v", hookType.String(), err)
		log.Printf("Expected JSON format for %s: %s", hookType.String(), GetExpectedJSONFormat(hookType.String()))
		h.respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Validation error: %v", err))
		return
	}

	// Convert validated request back to legacy payload format for domain layer
	payload := h.convertToLegacyPayload(validatedReq)

	// Extract hook type from Claude Code JSON (prefer hook_event_name over our URL-based hookType)
	if claudeHookType, ok := payload["hook_event_name"]; ok {
		if hookTypeStr, ok := claudeHookType.(string); ok {
			if parsedType, err := domain.ParseHookType(hookTypeStr); err == nil {
				hookType = parsedType
			}
		}
	}

	// Create structured hook data
	hookData := domain.NewHookData(hookType, payload)

	// Log received Claude Code data for debugging
	log.Printf("Received Claude Code hook data: session_id=%s, tool_name=%s", 
		hookData.GetSessionID(), hookData.GetToolName())

	// Create task and wait for user decision (5 minute timeout)
	log.Printf("Creating task and waiting for user decision for hook: %s", hookType.String())
	hookResponse, err := h.taskService.CreateTaskAndWaitForDecision(r.Context(), hookData, 5*time.Minute)
	if err != nil {
		log.Printf("Failed to get user decision for hook: %v", err)
		h.respondWithError(w, http.StatusInternalServerError, "Failed to process blocking webhook")
		return
	}

	// Log the response type
	log.Printf("Hook response for %s: %s", hookType.String(), hookResponse.String())

	// Send Claude Code compliant JSON response
	h.respondWithJSON(w, http.StatusOK, hookResponse)
}

// respondWithJSON sends a JSON response
func (h *WebhookHandler) respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}
