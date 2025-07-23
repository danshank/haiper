package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// NotificationPriority represents the urgency level of a notification
type NotificationPriority string

const (
	PriorityLow    NotificationPriority = "low"
	PriorityNormal NotificationPriority = "normal"
	PriorityHigh   NotificationPriority = "high"
	PriorityUrgent NotificationPriority = "urgent"
)

// Notification represents a push notification to be sent to the user
type Notification struct {
	ID          uuid.UUID            `json:"id"`
	TaskID      uuid.UUID            `json:"task_id"`
	Title       string               `json:"title"`
	Message     string               `json:"message"`
	Priority    NotificationPriority `json:"priority"`
	ActionURL   string               `json:"action_url"`   // URL to task management page
	Tags        []string             `json:"tags"`
	CreatedAt   time.Time            `json:"created_at"`
	SentAt      *time.Time           `json:"sent_at,omitempty"`
	DeliveredAt *time.Time           `json:"delivered_at,omitempty"`
}

// NewNotification creates a new notification for a task
func NewNotification(taskID uuid.UUID, hookType HookType, webDomain string) *Notification {
	notification := &Notification{
		ID:        uuid.New(),
		TaskID:    taskID,
		Priority:  PriorityNormal,
		Tags:      []string{"claude-code"},
		CreatedAt: time.Now(),
		ActionURL: fmt.Sprintf("http://%s/task/%s", webDomain, taskID.String()),
	}
	
	// Set title and message based on hook type
	switch hookType {
	case HookTypePreToolUse:
		notification.Title = "🔧 Claude Code - Tool Approval"
		notification.Message = "Claude needs permission to execute a tool"
		notification.Priority = PriorityHigh
		notification.Tags = append(notification.Tags, "tool-approval")
		
	case HookTypeNotification:
		notification.Title = "⚠️ Claude Code - Attention Required"
		notification.Message = "Claude Code needs your attention"
		notification.Priority = PriorityHigh
		notification.Tags = append(notification.Tags, "attention")
		
	case HookTypeUserPromptSubmit:
		notification.Title = "📝 Claude Code - Prompt Validation"
		notification.Message = "New prompt submitted for validation"
		notification.Priority = PriorityNormal
		notification.Tags = append(notification.Tags, "prompt")
		
	case HookTypePostToolUse:
		notification.Title = "✅ Claude Code - Tool Completed"
		notification.Message = "Tool execution completed"
		notification.Priority = PriorityLow
		notification.Tags = append(notification.Tags, "completed")
		
	case HookTypeStop:
		notification.Title = "🏁 Claude Code - Session Complete"
		notification.Message = "Claude Code session has finished"
		notification.Priority = PriorityLow
		notification.Tags = append(notification.Tags, "finished")
		
	case HookTypeSubagentStop:
		notification.Title = "🤖 Claude Code - Subagent Complete"
		notification.Message = "Claude Code subagent has finished"
		notification.Priority = PriorityLow
		notification.Tags = append(notification.Tags, "subagent")
		
	case HookTypePreCompact:
		notification.Title = "🗜️ Claude Code - Compacting"
		notification.Message = "Claude Code is compacting context"
		notification.Priority = PriorityNormal
		notification.Tags = append(notification.Tags, "compact")
		
	default:
		notification.Title = "🔔 Claude Code - Event"
		notification.Message = fmt.Sprintf("Hook event: %s", hookType.String())
	}
	
	return notification
}

// MarkSent records when the notification was sent
func (n *Notification) MarkSent() {
	now := time.Now()
	n.SentAt = &now
}

// MarkDelivered records when the notification was delivered
func (n *Notification) MarkDelivered() {
	now := time.Now()
	n.DeliveredAt = &now
}

// IsSent returns true if the notification has been sent
func (n *Notification) IsSent() bool {
	return n.SentAt != nil
}

// IsDelivered returns true if the notification has been delivered
func (n *Notification) IsDelivered() bool {
	return n.DeliveredAt != nil
}