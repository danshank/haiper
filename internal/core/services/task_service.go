package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dan/claude-control/internal/core/domain"
	"github.com/dan/claude-control/internal/core/ports"
	"github.com/google/uuid"
)

// TaskService handles the core business logic for task management
type TaskService struct {
	taskRepo        ports.TaskRepository
	historyRepo     ports.TaskHistoryRepository
	notificationSvc ports.NotificationSender
	responseBuilder ports.HookResponseBuilder
	decisionManager *TaskDecisionManager
	config          *TaskServiceConfig
}

// TaskServiceConfig holds configuration for the task service
type TaskServiceConfig struct {
	WebDomain          string `json:"web_domain"`
	AutoNotifyHookTypes []domain.HookType `json:"auto_notify_hook_types"`
}

// NewTaskService creates a new task service with dependencies
func NewTaskService(
	taskRepo ports.TaskRepository,
	historyRepo ports.TaskHistoryRepository,
	notificationSvc ports.NotificationSender,
	responseBuilder ports.HookResponseBuilder,
	config *TaskServiceConfig,
) *TaskService {
	if config.AutoNotifyHookTypes == nil {
		// Default hook types that should trigger notifications
		config.AutoNotifyHookTypes = []domain.HookType{
			domain.HookTypePreToolUse,
			domain.HookTypeNotification,
			domain.HookTypeUserPromptSubmit,
		}
	}

	return &TaskService{
		taskRepo:        taskRepo,
		historyRepo:     historyRepo,
		notificationSvc: notificationSvc,
		responseBuilder: responseBuilder,
		decisionManager: NewTaskDecisionManager(),
		config:          config,
	}
}

// CreateTask creates a new task with structured hook data
func (s *TaskService) CreateTask(ctx context.Context, task *domain.Task) error {
	// Store task
	if err := s.taskRepo.Create(ctx, task); err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Create history entry
	history := domain.NewTaskHistory(task.ID, domain.HistoryActionCreated, map[string]interface{}{
		"hook_type":  task.HookType.String(),
		"session_id": task.HookData.GetSessionID(),
		"tool_name":  task.HookData.GetToolName(),
	})
	if err := s.historyRepo.Create(ctx, history); err != nil {
		log.Printf("Warning: failed to create task history: %v", err)
		// Don't fail task creation due to history failure
	}

	// Send notification if configured for this hook type
	s.sendNotificationIfRequired(ctx, task)

	return nil
}

// sendNotificationIfRequired sends a notification if the hook type requires it
func (s *TaskService) sendNotificationIfRequired(ctx context.Context, task *domain.Task) {
	if s.shouldNotify(task.HookType) {
		if err := s.sendNotification(ctx, task); err != nil {
			log.Printf("Warning: failed to send notification for task %s: %v", task.ID, err)
		}
	}
}

// CreateTaskFromHook processes an incoming Claude Code hook and creates a task
func (s *TaskService) CreateTaskFromHook(ctx context.Context, hookData *domain.HookData) (*domain.Task, error) {
	// Create new task with structured data
	task := domain.NewTask(hookData)

	// Store task using the new CreateTask method
	if err := s.CreateTask(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

// GetTask retrieves a task by ID
func (s *TaskService) GetTask(ctx context.Context, taskID uuid.UUID) (*domain.Task, error) {
	return s.taskRepo.GetByID(ctx, taskID)
}

// GetTaskWithHistory retrieves a task and its history
func (s *TaskService) GetTaskWithHistory(ctx context.Context, taskID uuid.UUID) (*domain.Task, []*domain.TaskHistory, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, nil, err
	}

	history, err := s.historyRepo.GetByTaskID(ctx, taskID)
	if err != nil {
		return task, nil, fmt.Errorf("failed to get task history: %w", err)
	}

	return task, history, nil
}

// ListTasks retrieves tasks with filtering
func (s *TaskService) ListTasks(ctx context.Context, filter ports.TaskFilter) ([]*domain.Task, error) {
	return s.taskRepo.List(ctx, filter)
}

// GetPendingTasks retrieves all tasks that require user action
func (s *TaskService) GetPendingTasks(ctx context.Context) ([]*domain.Task, error) {
	return s.taskRepo.GetPendingTasks(ctx)
}

// TakeAction processes a user action on a task
func (s *TaskService) TakeAction(ctx context.Context, taskID uuid.UUID, action domain.ActionType, responseData map[string]interface{}) error {
	// Get the task
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Check if task is actionable
	if !task.IsActionable() {
		return fmt.Errorf("task %s is not actionable (status: %s)", taskID, task.Status.String())
	}

	// Take the action on the task
	task.TakeAction(action, responseData)

	// Update task in repository
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Create history entry
	history := domain.NewTaskHistory(task.ID, string(action), responseData)
	if err := s.historyRepo.Create(ctx, history); err != nil {
		log.Printf("Warning: failed to create task history: %v", err)
	}

	// Note: In JSON-based architecture, responses are handled via webhook returns
	// No need to send TMux commands as Claude Code receives JSON responses directly

	return nil
}

// sendNotification creates and sends a notification for a task
func (s *TaskService) sendNotification(ctx context.Context, task *domain.Task) error {
	notification := domain.NewNotification(task.ID, task.HookType, s.config.WebDomain)

	if err := s.notificationSvc.Send(ctx, notification); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	// Create history entry for notification
	history := domain.NewTaskHistory(task.ID, domain.HistoryActionNotified, map[string]interface{}{
		"notification_id": notification.ID.String(),
		"title":          notification.Title,
	})
	if err := s.historyRepo.Create(ctx, history); err != nil {
		log.Printf("Warning: failed to create notification history: %v", err)
	}

	return nil
}


// shouldNotify determines if a hook type should trigger a notification
func (s *TaskService) shouldNotify(hookType domain.HookType) bool {
	for _, notifyType := range s.config.AutoNotifyHookTypes {
		if hookType == notifyType {
			return true
		}
	}
	return false
}

// CreateTaskAndWaitForDecision creates a task and waits for user decision, returning hook response
func (s *TaskService) CreateTaskAndWaitForDecision(ctx context.Context, hookData *domain.HookData, timeout time.Duration) (*domain.HookResponse, error) {
	// Create new task with structured data
	task := domain.NewTask(hookData)

	// Store task
	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// Create history entry
	history := domain.NewTaskHistory(task.ID, domain.HistoryActionCreated, map[string]interface{}{
		"hook_type":  hookData.Type.String(),
		"session_id": hookData.GetSessionID(),
		"tool_name":  hookData.GetToolName(),
		"blocking":   true,
	})
	if err := s.historyRepo.Create(ctx, history); err != nil {
		log.Printf("Warning: failed to create task history: %v", err)
	}

	// Send notification if this hook type requires it
	if s.shouldNotify(hookData.Type) {
		if err := s.sendNotification(ctx, task); err != nil {
			log.Printf("Warning: failed to send notification for task %s: %v", task.ID, err)
		}
	}

	// Wait for user decision
	decision, err := s.decisionManager.WaitForDecision(ctx, task.ID.String(), timeout)
	if err != nil {
		// On timeout or error, update task status and return timeout response
		task.Status = domain.TaskStatusFailed
		s.taskRepo.Update(ctx, task)
		
		return s.responseBuilder.BuildTimeoutResponse(task.ID.String(), timeout), nil
	}

	// Update task with decision
	task.TakeAction(decision, map[string]interface{}{
		"decision_time": time.Now(),
		"blocking_call": true,
	})
	s.taskRepo.Update(ctx, task)

	// Create history entry for decision
	history = domain.NewTaskHistory(task.ID, string(decision), map[string]interface{}{
		"blocking_decision": true,
	})
	s.historyRepo.Create(ctx, history)

	// Return appropriate hook response based on user decision
	return s.responseBuilder.BuildResponseFromDecision(task.ID.String(), decision), nil
}

// CreateNonBlockingResponse creates a hook response for non-blocking hooks
func (s *TaskService) CreateNonBlockingResponse(ctx context.Context, hookData *domain.HookData, suppressOutput bool) (*domain.HookResponse, error) {
	// Create new task with structured data
	task := domain.NewTask(hookData)
	task.Status = domain.TaskStatusCompleted // Non-blocking tasks are immediately completed

	// Store task
	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// Create history entry
	history := domain.NewTaskHistory(task.ID, domain.HistoryActionCreated, map[string]interface{}{
		"hook_type":  hookData.Type.String(),
		"session_id": hookData.GetSessionID(),
		"tool_name":  hookData.GetToolName(),
		"blocking":   false,
	})
	if err := s.historyRepo.Create(ctx, history); err != nil {
		log.Printf("Warning: failed to create task history: %v", err)
	}

	// Send notification if this hook type requires it
	if s.shouldNotify(hookData.Type) {
		if err := s.sendNotification(ctx, task); err != nil {
			log.Printf("Warning: failed to send notification for task %s: %v", task.ID, err)
		}
	}

	// Return appropriate non-blocking response
	if suppressOutput {
		return s.responseBuilder.BuildSuppressedResponse(), nil
	}
	return s.responseBuilder.BuildContinueResponse(), nil
}

// SendDecisionToTask sends a user decision to a waiting task
func (s *TaskService) SendDecisionToTask(taskID uuid.UUID, decision domain.ActionType) bool {
	return s.decisionManager.SendDecision(taskID.String(), decision)
}

// HasPendingDecision checks if a task has a pending decision
func (s *TaskService) HasPendingDecision(taskID uuid.UUID) bool {
	return s.decisionManager.HasPendingDecision(taskID.String())
}

// GetActiveDecisions returns the number of active decision channels
func (s *TaskService) GetActiveDecisions() int {
	return s.decisionManager.GetActiveDecisions()
}

// CleanupOldTasks removes old completed tasks and their history
func (s *TaskService) CleanupOldTasks(ctx context.Context, retentionDays int) error {
	// This would typically be implemented with a database query
	// For now, we'll just clean up old history entries
	return s.historyRepo.DeleteOlderThan(ctx, retentionDays)
}