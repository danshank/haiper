package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TaskHistory represents an audit trail entry for task actions
type TaskHistory struct {
	ID        uuid.UUID              `json:"id"`
	TaskID    uuid.UUID              `json:"task_id"`
	Action    string                 `json:"action"`
	Data      map[string]interface{} `json:"data,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// NewTaskHistory creates a new task history entry
func NewTaskHistory(taskID uuid.UUID, action string, data map[string]interface{}) *TaskHistory {
	return &TaskHistory{
		ID:        uuid.New(),
		TaskID:    taskID,
		Action:    action,
		Data:      data,
		CreatedAt: time.Now(),
	}
}

// TaskHistoryAction constants for common actions
const (
	HistoryActionCreated   = "created"
	HistoryActionUpdated   = "updated"
	HistoryActionApproved  = "approved"
	HistoryActionRejected  = "rejected"
	HistoryActionCompleted = "completed"
	HistoryActionFailed    = "failed"
	HistoryActionNotified  = "notified"
)

// AddMetadata adds metadata to the history entry
func (h *TaskHistory) AddMetadata(key string, value interface{}) {
	if h.Data == nil {
		h.Data = make(map[string]interface{})
	}
	h.Data[key] = value
}

// GetDataAsJSON returns the data as JSON bytes
func (h *TaskHistory) GetDataAsJSON() ([]byte, error) {
	if h.Data == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(h.Data)
}