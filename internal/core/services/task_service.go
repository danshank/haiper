package services

import (
	"context"
	"encoding/json"
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
	tmuxController  ports.TMuxController
	decisionManager *TaskDecisionManager
	config          *TaskServiceConfig
}

// TaskServiceConfig holds configuration for the task service
type TaskServiceConfig struct {
	WebDomain          string `json:"web_domain"`
	TMuxSessionName    string `json:"tmux_session_name"`
	AutoNotifyHookTypes []domain.HookType `json:"auto_notify_hook_types"`
}

// NewTaskService creates a new task service with dependencies
func NewTaskService(
	taskRepo ports.TaskRepository,
	historyRepo ports.TaskHistoryRepository,
	notificationSvc ports.NotificationSender,
	tmuxController ports.TMuxController,
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
		tmuxController:  tmuxController,
		decisionManager: NewTaskDecisionManager(),
		config:          config,
	}
}

// CreateTaskFromHook processes an incoming Claude Code hook and creates a task
func (s *TaskService) CreateTaskFromHook(ctx context.Context, hookData *domain.HookData) (*domain.Task, error) {
	// Convert hook data to JSON
	taskDataBytes, err := json.Marshal(hookData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal hook data: %w", err)
	}

	// Create new task
	task := domain.NewTask(hookData.Type, json.RawMessage(taskDataBytes))

	// Store task
	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// Create history entry
	history := domain.NewTaskHistory(task.ID, domain.HistoryActionCreated, map[string]interface{}{
		"hook_type": hookData.Type.String(),
		"tool":      hookData.Tool,
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

	// Send command to tmux session
	if err := s.sendTMuxCommand(ctx, action); err != nil {
		log.Printf("Warning: failed to send tmux command: %v", err)
		// Don't return error here as the task action was successful
	}

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

// sendTMuxCommand sends the appropriate command to the tmux session
func (s *TaskService) sendTMuxCommand(ctx context.Context, action domain.ActionType) error {
	actionCmd := ports.GetActionCommand(action)

	if actionCmd.Command != "" {
		return s.tmuxController.SendCommand(ctx, s.config.TMuxSessionName, actionCmd.Command)
	}

	if actionCmd.Keys != "" {
		return s.tmuxController.SendKeys(ctx, s.config.TMuxSessionName, actionCmd.Keys)
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

// CreateTaskAndWaitForDecision creates a task and waits for user decision with timeout
func (s *TaskService) CreateTaskAndWaitForDecision(ctx context.Context, hookData *domain.HookData, timeout time.Duration) (*domain.Task, domain.ActionType, error) {
	// Convert hook data to JSON
	taskDataBytes, err := json.Marshal(hookData)
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal hook data: %w", err)
	}

	// Create new task
	task := domain.NewTask(hookData.Type, json.RawMessage(taskDataBytes))

	// Store task
	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, "", fmt.Errorf("failed to create task: %w", err)
	}

	// Create history entry
	history := domain.NewTaskHistory(task.ID, domain.HistoryActionCreated, map[string]interface{}{
		"hook_type": hookData.Type.String(),
		"tool":      hookData.Tool,
		"blocking":  true,
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
		// On timeout or error, update task status
		task.Status = domain.TaskStatusFailed
		s.taskRepo.Update(ctx, task)
		
		return task, "", err
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

	return task, decision, nil
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