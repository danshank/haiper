package ports

import (
	"context"
	"time"

	"github.com/dan/claude-control/internal/core/domain"
)

// TaskDecisionManager defines the interface for managing real-time decision channels for blocking webhook handlers
type TaskDecisionManager interface {
	// CreateDecisionChannel creates a new decision channel for a task
	CreateDecisionChannel(taskID string) chan domain.ActionType

	// SendDecision sends a decision to the waiting channel
	SendDecision(taskID string, decision domain.ActionType) bool

	// RemoveDecisionChannel removes and closes a decision channel
	RemoveDecisionChannel(taskID string)

	// WaitForDecision waits for a user decision with timeout
	WaitForDecision(ctx context.Context, taskID string, timeout time.Duration) (domain.ActionType, error)

	// GetActiveDecisions returns the number of active decision channels
	GetActiveDecisions() int

	// HasPendingDecision checks if a task has a pending decision
	HasPendingDecision(taskID string) bool

	// CleanupExpiredChannels removes channels that haven't been used (emergency cleanup)
	// This should rarely be needed as channels are cleaned up in defer statements
	CleanupExpiredChannels()
}
