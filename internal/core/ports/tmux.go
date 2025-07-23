package ports

import (
	"context"

	"github.com/dan/claude-control/internal/core/domain"
)

// TMuxController defines the interface for controlling TMux sessions
type TMuxController interface {
	// SendKeys sends keystrokes to a specific tmux session
	SendKeys(ctx context.Context, sessionName string, keys string) error
	
	// SendCommand sends a command to a tmux session (equivalent to typing + Enter)
	SendCommand(ctx context.Context, sessionName string, command string) error
	
	// ListSessions returns a list of available tmux sessions
	ListSessions(ctx context.Context) ([]TMuxSession, error)
	
	// SessionExists checks if a tmux session with the given name exists
	SessionExists(ctx context.Context, sessionName string) (bool, error)
	
	// CreateSession creates a new tmux session with the given name
	CreateSession(ctx context.Context, sessionName string) error
	
	// KillSession terminates a tmux session
	KillSession(ctx context.Context, sessionName string) error
	
	// GetSessionInfo retrieves detailed information about a session
	GetSessionInfo(ctx context.Context, sessionName string) (*TMuxSession, error)
}

// TMuxSession represents a tmux session
type TMuxSession struct {
	Name      string `json:"name"`
	Windows   int    `json:"windows"`
	Created   string `json:"created"`
	Attached  bool   `json:"attached"`
	LastUsed  string `json:"last_used"`
}

// TMuxConfig holds configuration for tmux integration
type TMuxConfig struct {
	DefaultSessionName string `json:"default_session_name"`
	SocketPath         string `json:"socket_path,omitempty"` // Optional custom socket path
}

// ActionCommand represents a command to send to Claude Code based on user action
type ActionCommand struct {
	Action  domain.ActionType `json:"action"`
	Command string           `json:"command"`
	Keys    string           `json:"keys"`
}

// GetActionCommand returns the appropriate tmux command for a user action
func GetActionCommand(action domain.ActionType) ActionCommand {
	switch action {
	case domain.ActionTypeApprove:
		return ActionCommand{
			Action:  action,
			Command: "",
			Keys:    "y Enter", // Send 'y' + Enter to approve
		}
	case domain.ActionTypeReject:
		return ActionCommand{
			Action:  action,
			Command: "",
			Keys:    "n Enter", // Send 'n' + Enter to reject
		}
	case domain.ActionTypeContinue:
		return ActionCommand{
			Action:  action,
			Command: "",
			Keys:    "Enter", // Just press Enter to continue
		}
	case domain.ActionTypeCancel:
		return ActionCommand{
			Action:  action,
			Command: "",
			Keys:    "C-c", // Send Ctrl+C to cancel
		}
	case domain.ActionTypeRetry:
		return ActionCommand{
			Action:  action,
			Command: "",
			Keys:    "r Enter", // Send 'r' + Enter to retry
		}
	default:
		return ActionCommand{
			Action:  action,
			Command: "",
			Keys:    "Enter",
		}
	}
}