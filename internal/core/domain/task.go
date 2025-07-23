package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusApproved  TaskStatus = "approved"
	TaskStatusRejected  TaskStatus = "rejected"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

func (s TaskStatus) String() string {
	return string(s)
}

func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusPending, TaskStatusApproved, TaskStatusRejected, TaskStatusCompleted, TaskStatusFailed:
		return true
	default:
		return false
	}
}

type ActionType string

const (
	ActionTypeApprove  ActionType = "approve"
	ActionTypeReject   ActionType = "reject"
	ActionTypeContinue ActionType = "continue"
	ActionTypeRetry    ActionType = "retry"
	ActionTypeCancel   ActionType = "cancel"
)

func (a ActionType) String() string {
	return string(a)
}

// Task represents a Claude Code task triggered by a webhook
type Task struct {
	ID           uuid.UUID              `json:"id"`
	HookType     HookType               `json:"hook_type"`
	TaskData     json.RawMessage        `json:"task_data"`     // Raw JSON from Claude Code hook
	Status       TaskStatus             `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	ActionTaken  *ActionType            `json:"action_taken,omitempty"`
	ResponseData map[string]interface{} `json:"response_data,omitempty"` // User's response/feedback
}

// NewTask creates a new task with default values
func NewTask(hookType HookType, taskData json.RawMessage) *Task {
	now := time.Now()
	return &Task{
		ID:        uuid.New(),
		HookType:  hookType,
		TaskData:  taskData,
		Status:    TaskStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// UpdateStatus updates the task status and timestamp
func (t *Task) UpdateStatus(status TaskStatus) {
	t.Status = status
	t.UpdatedAt = time.Now()
}

// TakeAction records an action taken on the task
func (t *Task) TakeAction(action ActionType, responseData map[string]interface{}) {
	t.ActionTaken = &action
	t.ResponseData = responseData
	t.UpdatedAt = time.Now()
	
	// Update status based on action
	switch action {
	case ActionTypeApprove:
		t.Status = TaskStatusApproved
	case ActionTypeReject:
		t.Status = TaskStatusRejected
	case ActionTypeContinue:
		t.Status = TaskStatusCompleted
	case ActionTypeCancel:
		t.Status = TaskStatusFailed
	}
}

// IsActionable returns true if the task can have actions taken on it
func (t *Task) IsActionable() bool {
	return t.Status == TaskStatusPending
}

// RequiresUserInput returns true if this hook type typically requires user interaction
func (t *Task) RequiresUserInput() bool {
	switch t.HookType {
	case HookTypeNotification, HookTypePreToolUse:
		return true
	case HookTypeUserPromptSubmit:
		return true // May need validation/blocking
	default:
		return false
	}
}