package ports

import (
	"context"

	"github.com/dan/claude-control/internal/core/domain"
	"github.com/google/uuid"
)

// TaskRepository defines the interface for task data persistence
type TaskRepository interface {
	// Create stores a new task
	Create(ctx context.Context, task *domain.Task) error
	
	// GetByID retrieves a task by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error)
	
	// Update updates an existing task
	Update(ctx context.Context, task *domain.Task) error
	
	// List retrieves tasks with optional filtering
	List(ctx context.Context, filter TaskFilter) ([]*domain.Task, error)
	
	// Delete removes a task by ID
	Delete(ctx context.Context, id uuid.UUID) error
	
	// GetPendingTasks retrieves all tasks that require user action
	GetPendingTasks(ctx context.Context) ([]*domain.Task, error)
	
	// GetTasksByHookType retrieves tasks filtered by hook type
	GetTasksByHookType(ctx context.Context, hookType domain.HookType) ([]*domain.Task, error)
}

// TaskHistoryRepository defines the interface for task history persistence
type TaskHistoryRepository interface {
	// Create stores a new task history entry
	Create(ctx context.Context, history *domain.TaskHistory) error
	
	// GetByTaskID retrieves all history entries for a task
	GetByTaskID(ctx context.Context, taskID uuid.UUID) ([]*domain.TaskHistory, error)
	
	// List retrieves history entries with optional filtering
	List(ctx context.Context, filter TaskHistoryFilter) ([]*domain.TaskHistory, error)
	
	// Delete removes history entries older than specified duration
	DeleteOlderThan(ctx context.Context, days int) error
}

// TaskFilter provides filtering options for task queries
type TaskFilter struct {
	Status    *domain.TaskStatus `json:"status,omitempty"`
	HookType  *domain.HookType   `json:"hook_type,omitempty"`
	Limit     int                `json:"limit,omitempty"`
	Offset    int                `json:"offset,omitempty"`
	SortBy    string             `json:"sort_by,omitempty"` // created_at, updated_at
	SortOrder string             `json:"sort_order,omitempty"` // asc, desc
}

// TaskHistoryFilter provides filtering options for task history queries
type TaskHistoryFilter struct {
	TaskID    *uuid.UUID `json:"task_id,omitempty"`
	Action    *string    `json:"action,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
	SortBy    string     `json:"sort_by,omitempty"` // created_at
	SortOrder string     `json:"sort_order,omitempty"` // asc, desc
}