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
	// Non-blocking webhook handlers (fire-and-forget)
	router.HandleFunc("/webhook/pre-tool-use", h.handlePreToolUse).Methods("POST")
	router.HandleFunc("/webhook/post-tool-use", h.handlePostToolUse).Methods("POST")
	router.HandleFunc("/webhook/notification", h.handleNotification).Methods("POST")
	router.HandleFunc("/webhook/user-prompt-submit", h.handleUserPromptSubmit).Methods("POST")
	router.HandleFunc("/webhook/stop", h.handleStop).Methods("POST")
	router.HandleFunc("/webhook/subagent-stop", h.handleSubagentStop).Methods("POST")
	router.HandleFunc("/webhook/pre-compact", h.handlePreCompact).Methods("POST")

	// Blocking webhook handlers (wait for user decision)
	router.HandleFunc("/webhook/pre-tool-use-blocking", h.handlePreToolUseBlocking).Methods("POST")
	router.HandleFunc("/webhook/notification-blocking", h.handleNotificationBlocking).Methods("POST")
	router.HandleFunc("/webhook/user-prompt-submit-blocking", h.handleUserPromptSubmitBlocking).Methods("POST")

	// Generic webhook handler for any hook type
	router.HandleFunc("/webhook/{hookType}", h.handleGenericWebhook).Methods("POST")
}

// handlePreToolUse handles PreToolUse webhook events
func (h *WebhookHandler) handlePreToolUse(w http.ResponseWriter, r *http.Request) {
	h.handleWebhook(w, r, domain.HookTypePreToolUse)
}

// handlePostToolUse handles PostToolUse webhook events
func (h *WebhookHandler) handlePostToolUse(w http.ResponseWriter, r *http.Request) {
	h.handleWebhook(w, r, domain.HookTypePostToolUse)
}

// handleNotification handles Notification webhook events
func (h *WebhookHandler) handleNotification(w http.ResponseWriter, r *http.Request) {
	h.handleWebhook(w, r, domain.HookTypeNotification)
}

// handleUserPromptSubmit handles UserPromptSubmit webhook events
func (h *WebhookHandler) handleUserPromptSubmit(w http.ResponseWriter, r *http.Request) {
	h.handleWebhook(w, r, domain.HookTypeUserPromptSubmit)
}

// handleStop handles Stop webhook events
func (h *WebhookHandler) handleStop(w http.ResponseWriter, r *http.Request) {
	h.handleWebhook(w, r, domain.HookTypeStop)
}

// handleSubagentStop handles SubagentStop webhook events
func (h *WebhookHandler) handleSubagentStop(w http.ResponseWriter, r *http.Request) {
	h.handleWebhook(w, r, domain.HookTypeSubagentStop)
}

// handlePreCompact handles PreCompact webhook events
func (h *WebhookHandler) handlePreCompact(w http.ResponseWriter, r *http.Request) {
	h.handleWebhook(w, r, domain.HookTypePreCompact)
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

	h.handleWebhook(w, r, hookType)
}

// handleWebhook is the core webhook handler logic
func (h *WebhookHandler) handleWebhook(w http.ResponseWriter, r *http.Request, hookType domain.HookType) {
	// Parse the incoming webhook payload
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	// Create hook data
	hookData := domain.NewHookData(hookType, payload)

	// Extract tool information if present
	if tool, ok := payload["tool"]; ok {
		if toolStr, ok := tool.(string); ok {
			hookData.Tool = toolStr
		}
	}

	// Extract matcher information if present (for PreCompact)
	if matcher, ok := payload["matcher"]; ok {
		if matcherStr, ok := matcher.(string); ok {
			hookData.Matcher = matcherStr
		}
	}

	// Create task from hook
	task, err := h.taskService.CreateTaskFromHook(r.Context(), hookData)
	if err != nil {
		log.Printf("Failed to create task from hook: %v", err)
		h.respondWithError(w, http.StatusInternalServerError, "Failed to process webhook")
		return
	}

	// Log the webhook event
	log.Printf("Processed %s webhook, created task %s", hookType.String(), task.ID.String())

	// Respond with the created task
	h.respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"success":  true,
		"task_id":  task.ID.String(),
		"message":  fmt.Sprintf("Webhook processed successfully for %s", hookType.String()),
		"task_url": fmt.Sprintf("/task/%s", task.ID.String()),
	})
}

// respondWithError sends an error response
func (h *WebhookHandler) respondWithError(w http.ResponseWriter, statusCode int, message string) {
	h.respondWithJSON(w, statusCode, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// handlePreToolUseBlocking handles blocking PreToolUse webhook events
func (h *WebhookHandler) handlePreToolUseBlocking(w http.ResponseWriter, r *http.Request) {
	h.handleBlockingWebhook(w, r, domain.HookTypePreToolUse)
}

// handleNotificationBlocking handles blocking Notification webhook events
func (h *WebhookHandler) handleNotificationBlocking(w http.ResponseWriter, r *http.Request) {
	h.handleBlockingWebhook(w, r, domain.HookTypeNotification)
}

// handleUserPromptSubmitBlocking handles blocking UserPromptSubmit webhook events
func (h *WebhookHandler) handleUserPromptSubmitBlocking(w http.ResponseWriter, r *http.Request) {
	h.handleBlockingWebhook(w, r, domain.HookTypeUserPromptSubmit)
}

// handleBlockingWebhook is the core blocking webhook handler logic
func (h *WebhookHandler) handleBlockingWebhook(w http.ResponseWriter, r *http.Request, hookType domain.HookType) {
	// Parse the incoming webhook payload
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	// Create hook data
	hookData := domain.NewHookData(hookType, payload)

	// Extract tool information if present
	if tool, ok := payload["tool"]; ok {
		if toolStr, ok := tool.(string); ok {
			hookData.Tool = toolStr
		}
	}

	// Extract matcher information if present (for PreCompact)
	if matcher, ok := payload["matcher"]; ok {
		if matcherStr, ok := matcher.(string); ok {
			hookData.Matcher = matcherStr
		}
	}

	// Create task and wait for user decision (5 minute timeout)
	task, decision, err := h.taskService.CreateTaskAndWaitForDecision(r.Context(), hookData, 5*time.Minute)
	if err != nil {
		log.Printf("Failed to get user decision for task: %v", err)

		// On timeout or error, apply default policy
		h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"continue":   false,
			"stopReason": fmt.Sprintf("Decision timeout or error: %v", err),
			"task_id":    task.ID.String(),
		})
		return
	}

	// Log the decision
	log.Printf("User decision for %s task %s: %s", hookType.String(), task.ID.String()[:8], decision)

	// Respond based on user decision
	switch decision {
	case domain.ActionTypeApprove:
		h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"continue": true,
			"task_id":  task.ID.String(),
			"message":  "User approved the action",
		})
	case domain.ActionTypeReject:
		h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"continue":   false,
			"stopReason": "User rejected the action",
			"task_id":    task.ID.String(),
		})
	case domain.ActionTypeCancel:
		h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"continue":   false,
			"stopReason": "User cancelled the action",
			"task_id":    task.ID.String(),
		})
	default:
		// Continue, retry, or other actions default to continue
		h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"continue": true,
			"task_id":  task.ID.String(),
			"message":  fmt.Sprintf("User action: %s", decision),
		})
	}
}

// respondWithJSON sends a JSON response
func (h *WebhookHandler) respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}
