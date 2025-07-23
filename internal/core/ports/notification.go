package ports

import (
	"context"

	"github.com/dan/claude-control/internal/core/domain"
)

// NotificationSender defines the interface for sending push notifications
type NotificationSender interface {
	// Send sends a notification and returns an error if delivery fails
	Send(ctx context.Context, notification *domain.Notification) error
	
	// SendBatch sends multiple notifications in a single operation
	SendBatch(ctx context.Context, notifications []*domain.Notification) error
	
	// Verify checks if the notification service is available and configured correctly
	Verify(ctx context.Context) error
}

// NotificationConfig holds configuration for notification services
type NotificationConfig struct {
	ServerURL string `json:"server_url"`
	Topic     string `json:"topic"`
	Token     string `json:"token,omitempty"`     // Optional authentication token
	Username  string `json:"username,omitempty"`  // Optional basic auth username
	Password  string `json:"password,omitempty"`  // Optional basic auth password
}