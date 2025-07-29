package domain

import (
	"time"

	"github.com/google/uuid"
)

type ActionType string

const (
	ActionTypeApprove      ActionType = "approve"
	ActionTypeReject       ActionType = "reject"
	ActionTypeSubmitPrompt ActionType = "submit_prompt"
)

// SessionAction represents an action taken in response to a session event
type SessionAction struct {
	ID             uuid.UUID  `json:"id"`
	SessionID      string     `json:"session_id"`
	SessionEventID uuid.UUID  `json:"session_event_id"`
	ActionType     ActionType `json:"action_type"`
	Input          string     `json:"input,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}
