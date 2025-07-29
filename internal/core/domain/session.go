package domain

import (
	"time"
)

// Session represents a Claude Code conversation session
type Session struct {
	ID        string    `json:"id"`         // Claude Code session ID
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
