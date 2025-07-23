package http

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/dan/claude-control/internal/core/domain"
	"github.com/dan/claude-control/internal/core/ports"
	"github.com/dan/claude-control/internal/core/services"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// WebHandler handles web interface requests
type WebHandler struct {
	taskService *services.TaskService
	templates   *template.Template
}

// NewWebHandler creates a new web handler
func NewWebHandler(taskService *services.TaskService) *WebHandler {
	return &WebHandler{
		taskService: taskService,
		templates:   template.Must(template.New("").ParseGlob("templates/*.html")),
	}
}

// RegisterRoutes registers web interface routes with the router
func (h *WebHandler) RegisterRoutes(router *mux.Router) {
	// Web interface routes
	router.HandleFunc("/", h.handleDashboard).Methods("GET")
	router.HandleFunc("/dashboard", h.handleDashboard).Methods("GET")
	router.HandleFunc("/task/{taskId}", h.handleTaskDetail).Methods("GET")
	router.HandleFunc("/task/{taskId}/action", h.handleTaskAction).Methods("POST")
	
	// API routes
	router.HandleFunc("/api/tasks", h.handleListTasks).Methods("GET")
	router.HandleFunc("/api/tasks/{taskId}", h.handleGetTask).Methods("GET")
	router.HandleFunc("/api/tasks/{taskId}/action", h.handleTaskActionAPI).Methods("POST")
	
	// Health check
	router.HandleFunc("/health", h.handleHealthCheck).Methods("GET")
}

// handleDashboard shows the main dashboard with pending tasks
func (h *WebHandler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Get pending tasks
	pendingTasks, err := h.taskService.GetPendingTasks(r.Context())
	if err != nil {
		log.Printf("Failed to get pending tasks: %v", err)
		http.Error(w, "Failed to load dashboard", http.StatusInternalServerError)
		return
	}

	// Get recent tasks
	recentFilter := ports.TaskFilter{
		Limit:     10,
		SortBy:    "created_at",
		SortOrder: "desc",
	}
	recentTasks, err := h.taskService.ListTasks(r.Context(), recentFilter)
	if err != nil {
		log.Printf("Failed to get recent tasks: %v", err)
		http.Error(w, "Failed to load dashboard", http.StatusInternalServerError)
		return
	}

	data := struct {
		PendingTasks []*domain.Task
		RecentTasks  []*domain.Task
		Title        string
	}{
		PendingTasks: pendingTasks,
		RecentTasks:  recentTasks,
		Title:        "Claude Control Dashboard",
	}

	if err := h.templates.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		log.Printf("Failed to render dashboard template: %v", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// handleTaskDetail shows detailed view of a specific task
func (h *WebHandler) handleTaskDetail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskIDStr := vars["taskId"]
	
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	// Get task with history
	task, history, err := h.taskService.GetTaskWithHistory(r.Context(), taskID)
	if err != nil {
		log.Printf("Failed to get task %s: %v", taskID, err)
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	data := struct {
		Task    *domain.Task
		History []*domain.TaskHistory
		Title   string
	}{
		Task:    task,
		History: history,
		Title:   fmt.Sprintf("Task %s", taskID.String()[:8]),
	}

	if err := h.templates.ExecuteTemplate(w, "task-detail.html", data); err != nil {
		log.Printf("Failed to render task detail template: %v", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// handleTaskAction processes user actions on tasks from web interface
func (h *WebHandler) handleTaskAction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskIDStr := vars["taskId"]
	
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	actionStr := r.FormValue("action")
	if actionStr == "" {
		http.Error(w, "Action is required", http.StatusBadRequest)
		return
	}

	action := domain.ActionType(actionStr)
	responseData := map[string]interface{}{
		"user_agent": r.Header.Get("User-Agent"),
		"timestamp":  r.FormValue("timestamp"),
		"comment":    r.FormValue("comment"),
	}

	// Check if this task has a pending decision (blocking webhook waiting)
	if h.taskService.HasPendingDecision(taskID) {
		// Send decision to waiting webhook handler
		success := h.taskService.SendDecisionToTask(taskID, action)
		if success {
			log.Printf("Sent decision %s to blocking webhook for task %s", action, taskID.String()[:8])
		} else {
			log.Printf("Warning: Failed to send decision to blocking webhook for task %s", taskID.String()[:8])
		}
	}

	// Take the action (update task in database)
	if err := h.taskService.TakeAction(r.Context(), taskID, action, responseData); err != nil {
		log.Printf("Failed to take action %s on task %s: %v", action, taskID, err)
		http.Error(w, "Failed to process action", http.StatusInternalServerError)
		return
	}

	// Redirect back to task detail page
	http.Redirect(w, r, fmt.Sprintf("/task/%s", taskID.String()), http.StatusSeeOther)
}

// handleListTasks returns tasks as JSON (API endpoint)
func (h *WebHandler) handleListTasks(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	filter := ports.TaskFilter{}
	
	if status := r.URL.Query().Get("status"); status != "" {
		taskStatus := domain.TaskStatus(status)
		if taskStatus.IsValid() {
			filter.Status = &taskStatus
		}
	}
	
	if hookType := r.URL.Query().Get("hook_type"); hookType != "" {
		if parsedHookType, err := domain.ParseHookType(hookType); err == nil {
			filter.HookType = &parsedHookType
		}
	}
	
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if parsedLimit, err := strconv.Atoi(limit); err == nil && parsedLimit > 0 {
			filter.Limit = parsedLimit
		}
	}
	
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if parsedOffset, err := strconv.Atoi(offset); err == nil && parsedOffset >= 0 {
			filter.Offset = parsedOffset
		}
	}

	tasks, err := h.taskService.ListTasks(r.Context(), filter)
	if err != nil {
		log.Printf("Failed to list tasks: %v", err)
		h.respondWithError(w, http.StatusInternalServerError, "Failed to list tasks")
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"tasks":   tasks,
		"count":   len(tasks),
	})
}

// handleGetTask returns a specific task as JSON (API endpoint)
func (h *WebHandler) handleGetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskIDStr := vars["taskId"]
	
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	task, history, err := h.taskService.GetTaskWithHistory(r.Context(), taskID)
	if err != nil {
		log.Printf("Failed to get task %s: %v", taskID, err)
		h.respondWithError(w, http.StatusNotFound, "Task not found")
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"task":    task,
		"history": history,
	})
}

// handleTaskActionAPI processes user actions on tasks via API
func (h *WebHandler) handleTaskActionAPI(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskIDStr := vars["taskId"]
	
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid task ID")
		return
	}

	// Parse JSON payload
	var payload struct {
		Action   string                 `json:"action"`
		Response map[string]interface{} `json:"response"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if payload.Action == "" {
		h.respondWithError(w, http.StatusBadRequest, "Action is required")
		return
	}

	action := domain.ActionType(payload.Action)
	responseData := payload.Response
	if responseData == nil {
		responseData = make(map[string]interface{})
	}

	// Add metadata
	responseData["user_agent"] = r.Header.Get("User-Agent")
	responseData["api_request"] = true

	// Check if this task has a pending decision (blocking webhook waiting)
	if h.taskService.HasPendingDecision(taskID) {
		// Send decision to waiting webhook handler
		success := h.taskService.SendDecisionToTask(taskID, action)
		if success {
			log.Printf("Sent decision %s to blocking webhook for task %s via API", action, taskID.String()[:8])
		} else {
			log.Printf("Warning: Failed to send decision to blocking webhook for task %s via API", taskID.String()[:8])
		}
	}

	// Take the action (update task in database)
	if err := h.taskService.TakeAction(r.Context(), taskID, action, responseData); err != nil {
		log.Printf("Failed to take action %s on task %s: %v", action, taskID, err)
		h.respondWithError(w, http.StatusInternalServerError, "Failed to process action")
		return
	}

	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Action %s processed successfully", action),
	})
}

// handleHealthCheck returns server health status
func (h *WebHandler) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status":  "healthy",
		"service": "claude-control-server",
	})
}

// respondWithError sends an error response
func (h *WebHandler) respondWithError(w http.ResponseWriter, statusCode int, message string) {
	h.respondWithJSON(w, statusCode, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// respondWithJSON sends a JSON response
func (h *WebHandler) respondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}