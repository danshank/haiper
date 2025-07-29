package ntfy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dan/claude-control/internal/core/domain"
	"github.com/dan/claude-control/internal/core/ports"
)

// NotificationSender implements the NotificationSender port for NTFY
type NotificationSender struct {
	config     *ports.NotificationConfig
	httpClient *http.Client
}

// NewNotificationSender creates a new NTFY notification sender
func NewNotificationSender(config *ports.NotificationConfig) *NotificationSender {
	return &NotificationSender{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Send sends a notification via NTFY
func (n *NotificationSender) Send(ctx context.Context, notification *domain.Notification) error {
	// Create NTFY message payload
	payload := map[string]interface{}{
		"topic":    n.config.Topic,
		"title":    notification.Title,
		"message":  notification.Message,
		"priority": n.mapPriority(notification.Priority),
		"tags":     notification.Tags,
		"click":    notification.ActionURL,
		"actions": []map[string]interface{}{
			{
				"action": "view",
				"label":  "Open Task",
				"url":    notification.ActionURL,
			},
		},
	}

	// Marshal payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal notification payload: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s", n.config.ServerURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Add authentication if configured
	if n.config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+n.config.Token)
	} else if n.config.Username != "" && n.config.Password != "" {
		req.SetBasicAuth(n.config.Username, n.config.Password)
	}

	// Send request
	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("NTFY server returned status %d", resp.StatusCode)
	}

	// Mark notification as sent
	notification.MarkSent()

	return nil
}

// Verify checks if the notification service is available and configured correctly
func (n *NotificationSender) Verify(ctx context.Context) error {
	// Create a simple health check request
	url := fmt.Sprintf("%s/v1/health", strings.TrimSuffix(n.config.ServerURL, "/"))
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	// Add authentication if configured
	if n.config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+n.config.Token)
	} else if n.config.Username != "" && n.config.Password != "" {
		req.SetBasicAuth(n.config.Username, n.config.Password)
	}

	// Send request
	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("NTFY server is not accessible: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("NTFY server health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// mapPriority converts domain notification priority to NTFY priority
func (n *NotificationSender) mapPriority(priority domain.NotificationPriority) int {
	switch priority {
	case domain.PriorityLow:
		return 2
	case domain.PriorityNormal:
		return 3
	case domain.PriorityHigh:
		return 4
	case domain.PriorityUrgent:
		return 5
	default:
		return 3 // Normal priority as default
	}
}