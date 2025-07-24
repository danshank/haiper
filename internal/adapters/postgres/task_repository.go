package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dan/claude-control/internal/core/domain"
	"github.com/dan/claude-control/internal/core/ports"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// TaskRepository implements the TaskRepository port for PostgreSQL
type TaskRepository struct {
	db *sql.DB
}

// NewTaskRepository creates a new PostgreSQL task repository
func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// Create stores a new task
func (r *TaskRepository) Create(ctx context.Context, task *domain.Task) error {
	query := `
		INSERT INTO tasks (id, hook_type, task_data, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	// Convert task data to string for PostgreSQL
	taskDataStr := string(task.TaskData)
	if taskDataStr == "" || taskDataStr == "null" {
		taskDataStr = "{}"
	}

	_, err := r.db.ExecContext(ctx, query,
		task.ID,
		task.HookType.String(),
		taskDataStr,
		task.Status.String(),
		task.CreatedAt,
		task.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}

// GetByID retrieves a task by its ID
func (r *TaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	query := `
		SELECT id, hook_type, task_data, status, created_at, updated_at, action_taken, response_data
		FROM tasks
		WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id)

	task, err := r.scanTask(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return task, nil
}

// Update updates an existing task
func (r *TaskRepository) Update(ctx context.Context, task *domain.Task) error {
	query := `
		UPDATE tasks
		SET hook_type = $2, task_data = $3, status = $4, updated_at = $5, action_taken = $6, response_data = $7
		WHERE id = $1`

	var actionTaken *string
	if task.ActionTaken != nil {
		action := task.ActionTaken.String()
		actionTaken = &action
	}

	var responseDataJSON []byte
	if task.ResponseData != nil {
		var err error
		responseDataJSON, err = json.Marshal(task.ResponseData)
		if err != nil {
			return fmt.Errorf("failed to marshal response data: %w", err)
		}
	}

	result, err := r.db.ExecContext(ctx, query,
		task.ID,
		task.HookType.String(),
		string(task.TaskData), // Convert json.RawMessage to string
		task.Status.String(),
		task.UpdatedAt,
		actionTaken,
		responseDataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found: %s", task.ID)
	}

	return nil
}

// List retrieves tasks with optional filtering
func (r *TaskRepository) List(ctx context.Context, filter ports.TaskFilter) ([]*domain.Task, error) {
	query := "SELECT id, hook_type, task_data, status, created_at, updated_at, action_taken, response_data FROM tasks"
	args := []interface{}{}
	conditions := []string{}
	argIndex := 1

	// Add WHERE conditions
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, filter.Status.String())
		argIndex++
	}

	if filter.HookType != nil {
		conditions = append(conditions, fmt.Sprintf("hook_type = $%d", argIndex))
		args = append(args, filter.HookType.String())
		argIndex++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add ORDER BY
	if filter.SortBy != "" {
		orderDirection := "ASC"
		if filter.SortOrder == "desc" {
			orderDirection = "DESC"
		}
		query += fmt.Sprintf(" ORDER BY %s %s", filter.SortBy, orderDirection)
	} else {
		query += " ORDER BY created_at DESC"
	}

	// Add LIMIT and OFFSET
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		task, err := r.scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tasks: %w", err)
	}

	return tasks, nil
}

// Delete removes a task by ID
func (r *TaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM tasks WHERE id = $1"

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task not found: %s", id)
	}

	return nil
}

// GetPendingTasks retrieves all tasks that require user action
func (r *TaskRepository) GetPendingTasks(ctx context.Context) ([]*domain.Task, error) {
	filter := ports.TaskFilter{
		Status: func() *domain.TaskStatus { s := domain.TaskStatusPending; return &s }(),
		SortBy: "created_at",
		SortOrder: "asc",
	}
	return r.List(ctx, filter)
}

// GetTasksByHookType retrieves tasks filtered by hook type
func (r *TaskRepository) GetTasksByHookType(ctx context.Context, hookType domain.HookType) ([]*domain.Task, error) {
	filter := ports.TaskFilter{
		HookType: &hookType,
		SortBy:   "created_at",
		SortOrder: "desc",
	}
	return r.List(ctx, filter)
}

// scanTask scans a database row into a Task struct
func (r *TaskRepository) scanTask(scanner interface {
	Scan(dest ...interface{}) error
}) (*domain.Task, error) {
	var task domain.Task
	var hookTypeStr, statusStr string
	var actionTakenStr *string
	var responseDataJSON []byte

	err := scanner.Scan(
		&task.ID,
		&hookTypeStr,
		&task.TaskData,
		&statusStr,
		&task.CreatedAt,
		&task.UpdatedAt,
		&actionTakenStr,
		&responseDataJSON,
	)

	if err != nil {
		return nil, err
	}

	// Parse hook type
	hookType, err := domain.ParseHookType(hookTypeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid hook type in database: %s", hookTypeStr)
	}
	task.HookType = hookType

	// Parse status
	task.Status = domain.TaskStatus(statusStr)
	if !task.Status.IsValid() {
		return nil, fmt.Errorf("invalid status in database: %s", statusStr)
	}

	// Parse action taken
	if actionTakenStr != nil {
		actionType := domain.ActionType(*actionTakenStr)
		task.ActionTaken = &actionType
	}

	// Parse response data
	if responseDataJSON != nil {
		var responseData map[string]interface{}
		if err := json.Unmarshal(responseDataJSON, &responseData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response data: %w", err)
		}
		task.ResponseData = responseData
	}

	return &task, nil
}