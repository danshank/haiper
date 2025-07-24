package http

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dan/claude-control/internal/core/domain"
	"github.com/dan/claude-control/internal/core/services"
	"github.com/gorilla/mux"
)

// WebhookHandler handles Claude Code webhook requests
type WebhookHandler struct {
	taskService *services.TaskService
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(taskService *services.TaskService) *WebhookHandler {
	return &WebhookHandler{
		taskService: taskService,
	}
}

// RegisterRoutes registers webhook routes with the router
func (h *WebhookHandler) RegisterRoutes(router *mux.Router) {
	// Blocking webhook handlers (require user approval)
	router.HandleFunc("/webhook/notification", h.handleNotificationBlocking).Methods("POST")
	router.HandleFunc("/webhook/stop", h.handleStopBlocking).Methods("POST")
	router.HandleFunc("/webhook/subagent-stop", h.handleSubagentStopBlocking).Methods("POST")

	// Non-blocking webhook handlers (immediate response, no task creation)
	router.HandleFunc("/webhook/pre-tool-use", h.handlePreToolUse).Methods("POST")
	router.HandleFunc("/webhook/post-tool-use", h.handlePostToolUse).Methods("POST")
	router.HandleFunc("/webhook/user-prompt-submit", h.handleUserPromptSubmit).Methods("POST")
	router.HandleFunc("/webhook/pre-compact", h.handlePreCompact).Methods("POST")

	// Generic webhook handler for any hook type
	router.HandleFunc("/webhook/{hookType}", h.handleGenericWebhook).Methods("POST")
}

// handlePostToolUse handles PostToolUse webhook events
func (h *WebhookHandler) handlePostToolUse(w http.ResponseWriter, r *http.Request) {
	h.handleNonBlockingWebhook(w, r, domain.HookTypePostToolUse)
}

// handleStopBlocking handles blocking Stop webhook events
func (h *WebhookHandler) handleStopBlocking(w http.ResponseWriter, r *http.Request) {
	h.handleBlockingWebhook(w, r, domain.HookTypeStop)
}

// handleSubagentStopBlocking handles blocking SubagentStop webhook events
func (h *WebhookHandler) handleSubagentStopBlocking(w http.ResponseWriter, r *http.Request) {
	h.handleBlockingWebhook(w, r, domain.HookTypeSubagentStop)
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
		h.respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid hook type: %s", hookTypeStr))
		return
	}

	h.handleNonBlockingWebhook(w, r, hookType)
}

// handleNonBlockingWebhook handles webhooks that don't require user approval
func (h *WebhookHandler) handleNonBlockingWebhook(w http.ResponseWriter, r *http.Request, hookType domain.HookType) {
	// Parse the incoming webhook payload with detailed error logging
	var payload map[string]interface{}
	if err := DecodeJSONWithDebug(r, &payload, 1024*1024); err != nil { // 1MB limit
		log.Printf("Expected JSON format for %s: %s", hookType.String(), GetExpectedJSONFormat(hookType.String()))
		h.respondWithError(w, http.StatusBadRequest, fmt.Sprintf("JSON parsing failed: %v", err))
		return
	}

	// Extract hook type from Claude Code JSON (prefer hook_event_name over our URL-based hookType)
	if claudeHookType, ok := payload["hook_event_name"]; ok {
		if hookTypeStr, ok := claudeHookType.(string); ok {
			if parsedType, err := domain.ParseHookType(hookTypeStr); err == nil {
				hookType = parsedType
			}
		}
	}

	// Create hook data with full Claude Code payload
	hookData := domain.NewHookData(hookType, payload)

	// Extract tool information from Claude Code format
	if tool, ok := payload["tool_name"]; ok {
		if toolStr, ok := tool.(string); ok {
			hookData.Tool = toolStr
		}
	} else if tool, ok := payload["tool"]; ok {
		if toolStr, ok := tool.(string); ok {
			hookData.Tool = toolStr
		}
	}

	// Extract matcher information for PreCompact
	if matcher, ok := payload["matcher"]; ok {
		if matcherStr, ok := matcher.(string); ok {
			hookData.Matcher = matcherStr
		}
	}

	// Log received Claude Code data for debugging
	log.Printf("Received Claude Code hook data: session_id=%v, cwd=%v, tool_name=%v", 
		payload["session_id"], payload["cwd"], payload["tool_name"])

	// Return immediate non-blocking response (no task creation needed)
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

// handleNotificationBlocking handles blocking Notification webhook events
func (h *WebhookHandler) handleNotificationBlocking(w http.ResponseWriter, r *http.Request) {
	h.handleBlockingWebhook(w, r, domain.HookTypeNotification)
}

// handleUserPromptSubmit handles non-blocking UserPromptSubmit webhook events
func (h *WebhookHandler) handleUserPromptSubmit(w http.ResponseWriter, r *http.Request) {
	h.handleNonBlockingWebhook(w, r, domain.HookTypeUserPromptSubmit)
}

// handleBlockingWebhook is the core blocking webhook handler logic
func (h *WebhookHandler) handleBlockingWebhook(w http.ResponseWriter, r *http.Request, hookType domain.HookType) {
	// Parse the incoming webhook payload with detailed error logging
	var payload map[string]interface{}
	if err := DecodeJSONWithDebug(r, &payload, 1024*1024); err != nil { // 1MB limit
		log.Printf("Expected JSON format for %s: %s", hookType.String(), GetExpectedJSONFormat(hookType.String()))
		h.respondWithError(w, http.StatusBadRequest, fmt.Sprintf("JSON parsing failed: %v", err))
		return
	}

	// Extract hook type from Claude Code JSON (prefer hook_event_name over our URL-based hookType)
	if claudeHookType, ok := payload["hook_event_name"]; ok {
		if hookTypeStr, ok := claudeHookType.(string); ok {
			if parsedType, err := domain.ParseHookType(hookTypeStr); err == nil {
				hookType = parsedType
			}
		}
	}

	// Create hook data with full Claude Code payload
	hookData := domain.NewHookData(hookType, payload)

	// Extract tool information from Claude Code format
	if tool, ok := payload["tool_name"]; ok {
		if toolStr, ok := tool.(string); ok {
			hookData.Tool = toolStr
		}
	} else if tool, ok := payload["tool"]; ok {
		if toolStr, ok := tool.(string); ok {
			hookData.Tool = toolStr
		}
	}

	// Extract matcher information for PreCompact
	if matcher, ok := payload["matcher"]; ok {
		if matcherStr, ok := matcher.(string); ok {
			hookData.Matcher = matcherStr
		}
	}

	// Log received Claude Code data for debugging
	log.Printf("Received Claude Code hook data: session_id=%v, cwd=%v, tool_name=%v", 
		payload["session_id"], payload["cwd"], payload["tool_name"])

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
