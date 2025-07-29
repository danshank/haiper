package ports

import (
	"context"
	"time"

	"github.com/dan/claude-control/internal/core/domain"
	"github.com/google/uuid"
)

// TaskService defines the interface for task management business logic
type TaskService interface {
	// CreateTask creates a new task with structured hook data
	CreateTask(ctx context.Context, task *domain.Task) error

	// CreateTaskFromHook processes an incoming Claude Code hook and creates a task
	CreateTaskFromHook(ctx context.Context, hookData *domain.HookData) (*domain.Task, error)

	// GetTask retrieves a task by ID
	GetTask(ctx context.Context, taskID uuid.UUID) (*domain.Task, error)

	// GetTaskWithHistory retrieves a task and its history
	GetTaskWithHistory(ctx context.Context, taskID uuid.UUID) (*domain.Task, []*domain.TaskHistory, error)

	// ListTasks retrieves tasks with filtering
	ListTasks(ctx context.Context, filter TaskFilter) ([]*domain.Task, error)

	// GetPendingTasks retrieves all tasks that require user action
	GetPendingTasks(ctx context.Context) ([]*domain.Task, error)

	// TakeAction processes a user action on a task
	TakeAction(ctx context.Context, taskID uuid.UUID, action domain.ActionType, responseData map[string]interface{}) error

	// CreateTaskAndWaitForDecision creates a task and waits for user decision, returning hook response
	CreateTaskAndWaitForDecision(ctx context.Context, hookData *domain.HookData, timeout time.Duration) (*domain.HookResponse, error)

	// CreateNonBlockingResponse creates a hook response for non-blocking hooks
	CreateNonBlockingResponse(ctx context.Context, hookData *domain.HookData, suppressOutput bool) (*domain.HookResponse, error)

	// SendDecisionToTask sends a user decision to a waiting task
	SendDecisionToTask(taskID uuid.UUID, decision domain.ActionType) bool

	// HasPendingDecision checks if a task has a pending decision
	HasPendingDecision(taskID uuid.UUID) bool

	// GetActiveDecisions returns the number of active decision channels
	GetActiveDecisions() int

	// CleanupOldTasks removes old completed tasks and their history
	CleanupOldTasks(ctx context.Context, retentionDays int) error
}
