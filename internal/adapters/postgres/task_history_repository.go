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
)

// TaskHistoryRepository implements the TaskHistoryRepository port for PostgreSQL
type TaskHistoryRepository struct {
	db *sql.DB
}

// NewTaskHistoryRepository creates a new PostgreSQL task history repository
func NewTaskHistoryRepository(db *sql.DB) *TaskHistoryRepository {
	return &TaskHistoryRepository{db: db}
}

// Create stores a new task history entry
func (r *TaskHistoryRepository) Create(ctx context.Context, history *domain.TaskHistory) error {
	query := `
		INSERT INTO task_history (id, task_id, action, data, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	var dataJSON []byte
	if history.Data != nil {
		var err error
		dataJSON, err = json.Marshal(history.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal history data: %w", err)
		}
	}

	_, err := r.db.ExecContext(ctx, query,
		history.ID,
		history.TaskID,
		history.Action,
		dataJSON,
		history.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create task history: %w", err)
	}

	return nil
}

// GetByTaskID retrieves all history entries for a task
func (r *TaskHistoryRepository) GetByTaskID(ctx context.Context, taskID uuid.UUID) ([]*domain.TaskHistory, error) {
	query := `
		SELECT id, task_id, action, data, created_at
		FROM task_history
		WHERE task_id = $1
		ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task history: %w", err)
	}
	defer rows.Close()

	var histories []*domain.TaskHistory
	for rows.Next() {
		history, err := r.scanTaskHistory(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task history: %w", err)
		}
		histories = append(histories, history)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task history: %w", err)
	}

	return histories, nil
}

// List retrieves history entries with optional filtering
func (r *TaskHistoryRepository) List(ctx context.Context, filter ports.TaskHistoryFilter) ([]*domain.TaskHistory, error) {
	query := "SELECT id, task_id, action, data, created_at FROM task_history"
	args := []interface{}{}
	conditions := []string{}
	argIndex := 1

	// Add WHERE conditions
	if filter.TaskID != nil {
		conditions = append(conditions, fmt.Sprintf("task_id = $%d", argIndex))
		args = append(args, *filter.TaskID)
		argIndex++
	}

	if filter.Action != nil {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argIndex))
		args = append(args, *filter.Action)
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
		return nil, fmt.Errorf("failed to list task history: %w", err)
	}
	defer rows.Close()

	var histories []*domain.TaskHistory
	for rows.Next() {
		history, err := r.scanTaskHistory(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task history: %w", err)
		}
		histories = append(histories, history)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task history: %w", err)
	}

	return histories, nil
}

// DeleteOlderThan removes history entries older than specified duration
func (r *TaskHistoryRepository) DeleteOlderThan(ctx context.Context, days int) error {
	query := `
		DELETE FROM task_history
		WHERE created_at < NOW() - INTERVAL '%d days'`

	result, err := r.db.ExecContext(ctx, fmt.Sprintf(query, days))
	if err != nil {
		return fmt.Errorf("failed to delete old task history: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Log the number of deleted rows (you might want to use a proper logger)
	fmt.Printf("Deleted %d old task history entries\n", rowsAffected)

	return nil
}

// scanTaskHistory scans a database row into a TaskHistory struct
func (r *TaskHistoryRepository) scanTaskHistory(scanner interface {
	Scan(dest ...interface{}) error
}) (*domain.TaskHistory, error) {
	var history domain.TaskHistory
	var dataJSON []byte

	err := scanner.Scan(
		&history.ID,
		&history.TaskID,
		&history.Action,
		&dataJSON,
		&history.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Parse data
	if dataJSON != nil {
		var data map[string]interface{}
		if err := json.Unmarshal(dataJSON, &data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal history data: %w", err)
		}
		history.Data = data
	}

	return &history, nil
}