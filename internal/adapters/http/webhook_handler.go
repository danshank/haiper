package http

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/dan/claude-control/internal/core/domain"
	"github.com/dan/claude-control/internal/core/ports"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// WebhookHandler handles Claude Code webhook requests with validation
type WebhookHandler struct {
	sessionService ports.SessionService
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(sessionService ports.SessionService) *WebhookHandler {
	return &WebhookHandler{
		sessionService: sessionService,
	}
}

// RegisterRoutes registers webhook routes with the router
func (h *WebhookHandler) RegisterRoutes(router *mux.Router) {
	// Generic webhook handler for all hook types
	router.HandleFunc("/webhook/{hookType}", h.handleWebhook).Methods("POST")
}

// parseSessionEvent parses the incoming webhook request directly into a session event
func (h *WebhookHandler) parseSessionEvent(r *http.Request, hookType domain.HookType) (*domain.SessionEvent, error) {
	var rawData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&rawData); err != nil {
		return nil, err
	}

	// Extract required fields
	sessionID, _ := rawData["session_id"].(string)
	cwd, _ := rawData["cwd"].(string)
	transcriptPath, _ := rawData["transcript_path"].(string)

	return &domain.SessionEvent{
		ID:             uuid.New(),
		SessionID:      sessionID,
		HookType:       hookType,
		CWD:            cwd,
		TranscriptPath: transcriptPath,
		EventData:      rawData,
		CreatedAt:      time.Now(),
	}, nil
}

// respondWithJSON sends a JSON response
func (h *WebhookHandler) respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// handleWebhook handles all webhook events generically
func (h *WebhookHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hookTypeStr := vars["hookType"]
	
	hookType, err := domain.ParseHookType(hookTypeStr)
	if err != nil {
		log.Printf("Unknown hook type: %s", hookTypeStr)
		h.respondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Unknown hook type"})
		return
	}

	event, err := h.parseSessionEvent(r, hookType)
	if err != nil {
		log.Printf("Failed to parse %s webhook: %v", hookTypeStr, err)
		h.respondWithJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if err := h.sessionService.AppendEvent(r.Context(), event.SessionID, event); err != nil {
		log.Printf("Failed to append %s event: %v", hookTypeStr, err)
	}

	// Determine response based on hook type
	suppressOutput := hookType == domain.HookTypeStop || hookType == domain.HookTypeSubagentStop
	h.respondWithJSON(w, http.StatusOK, &domain.HookResponse{Continue: true, SuppressOutput: suppressOutput})
}
