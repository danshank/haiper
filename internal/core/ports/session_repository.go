package ports

import (
	"context"

	"github.com/dan/claude-control/internal/core/domain"
)

// SessionRepository defines the interface for session data persistence
type SessionRepository interface {
	// GetSession retrieves a session by its ID, and creates it if it doesn't exist
	GetSession(ctx context.Context, sessionID string) (*domain.Session, error)

	// AddEvent stores a new event for a session
	AddEvent(ctx context.Context, sessionID string, event *domain.SessionEvent) error

	// GetEvents retrieves events for a session with optional filtering
	GetEvents(ctx context.Context, sessionID string, filter EventFilter) ([]*domain.SessionEvent, error)
}