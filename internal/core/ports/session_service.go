package ports

import (
	"context"

	"github.com/dan/claude-control/internal/core/domain"
)

// SessionService defines the interface for session management business logic
type SessionService interface {
	// GetOrCreateSession retrieves an existing session or creates a new one
	GetOrCreateSession(ctx context.Context, sessionID string) (*domain.Session, error)

	// AppendEvent adds a new event to a session
	AppendEvent(ctx context.Context, sessionID string, event *domain.SessionEvent) error

	// GetSessionEvents retrieves events for a session with optional filtering
	GetSessionEvents(ctx context.Context, sessionID string, filter EventFilter) ([]*domain.SessionEvent, error)
}

// EventFilter provides filtering options for session event queries
type EventFilter struct {
	HookType  *domain.HookType `json:"hook_type,omitempty"`
	Limit     int              `json:"limit,omitempty"`
	Offset    int              `json:"offset,omitempty"`
	SortBy    string           `json:"sort_by,omitempty"`    // created_at
	SortOrder string           `json:"sort_order,omitempty"` // asc, desc
}