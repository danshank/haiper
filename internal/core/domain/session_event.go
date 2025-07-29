package domain

import (
	"time"

	"github.com/google/uuid"
)

// SessionEvent represents a single hook event within a Claude Code session
type SessionEvent struct {
	ID             uuid.UUID   `json:"id"`
	SessionID      string      `json:"session_id"`
	HookType       HookType    `json:"hook_type"`        // Parsed hook type
	CWD            string      `json:"cwd"`              // Working directory for this event
	TranscriptPath string      `json:"transcript_path"`  // Transcript path for this event
	EventData      interface{} `json:"event_data"`       // Hook-specific data
	CreatedAt      time.Time   `json:"created_at"`
}
