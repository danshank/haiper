package tmux

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/dan/claude-control/internal/core/ports"
)

// Controller implements the TMuxController port
type Controller struct {
	config *ports.TMuxConfig
}

// NewController creates a new TMux controller
func NewController(config *ports.TMuxConfig) *Controller {
	return &Controller{
		config: config,
	}
}

// SendKeys sends keystrokes to a specific tmux session
func (c *Controller) SendKeys(ctx context.Context, sessionName string, keys string) error {
	args := []string{"send-keys", "-t", sessionName}
	
	// Add socket path if configured
	if c.config.SocketPath != "" {
		args = append([]string{"-S", c.config.SocketPath}, args...)
	}
	
	// Split keys by spaces and add each as separate argument
	keyParts := strings.Fields(keys)
	args = append(args, keyParts...)

	cmd := exec.CommandContext(ctx, "tmux", args...)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to send keys to tmux session %s: %w (output: %s)", sessionName, err, string(output))
	}

	return nil
}

// SendCommand sends a command to a tmux session (equivalent to typing + Enter)
func (c *Controller) SendCommand(ctx context.Context, sessionName string, command string) error {
	// First send the command text
	if err := c.SendKeys(ctx, sessionName, fmt.Sprintf("'%s'", command)); err != nil {
		return err
	}
	
	// Then send Enter to execute it
	return c.SendKeys(ctx, sessionName, "Enter")
}

// ListSessions returns a list of available tmux sessions
func (c *Controller) ListSessions(ctx context.Context) ([]ports.TMuxSession, error) {
	args := []string{"list-sessions", "-F", "#{session_name}:#{session_windows}:#{session_created}:#{session_attached}:#{session_last_attached}"}
	
	// Add socket path if configured
	if c.config.SocketPath != "" {
		args = append([]string{"-S", c.config.SocketPath}, args...)
	}

	cmd := exec.CommandContext(ctx, "tmux", args...)
	
	output, err := cmd.Output()
	if err != nil {
		// If no sessions exist, tmux returns exit code 1
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return []ports.TMuxSession{}, nil
		}
		return nil, fmt.Errorf("failed to list tmux sessions: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	sessions := make([]ports.TMuxSession, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}
		
		parts := strings.Split(line, ":")
		if len(parts) < 5 {
			continue
		}

		windows, _ := strconv.Atoi(parts[1])
		
		session := ports.TMuxSession{
			Name:     parts[0],
			Windows:  windows,
			Created:  c.formatTimestamp(parts[2]),
			Attached: parts[3] == "1",
			LastUsed: c.formatTimestamp(parts[4]),
		}
		
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// SessionExists checks if a tmux session with the given name exists
func (c *Controller) SessionExists(ctx context.Context, sessionName string) (bool, error) {
	args := []string{"has-session", "-t", sessionName}
	
	// Add socket path if configured
	if c.config.SocketPath != "" {
		args = append([]string{"-S", c.config.SocketPath}, args...)
	}

	cmd := exec.CommandContext(ctx, "tmux", args...)
	
	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			// Session doesn't exist
			return false, nil
		}
		return false, fmt.Errorf("failed to check if tmux session exists: %w", err)
	}

	return true, nil
}

// CreateSession creates a new tmux session with the given name
func (c *Controller) CreateSession(ctx context.Context, sessionName string) error {
	args := []string{"new-session", "-d", "-s", sessionName}
	
	// Add socket path if configured
	if c.config.SocketPath != "" {
		args = append([]string{"-S", c.config.SocketPath}, args...)
	}

	cmd := exec.CommandContext(ctx, "tmux", args...)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create tmux session %s: %w (output: %s)", sessionName, err, string(output))
	}

	return nil
}

// KillSession terminates a tmux session
func (c *Controller) KillSession(ctx context.Context, sessionName string) error {
	args := []string{"kill-session", "-t", sessionName}
	
	// Add socket path if configured
	if c.config.SocketPath != "" {
		args = append([]string{"-S", c.config.SocketPath}, args...)
	}

	cmd := exec.CommandContext(ctx, "tmux", args...)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to kill tmux session %s: %w (output: %s)", sessionName, err, string(output))
	}

	return nil
}

// GetSessionInfo retrieves detailed information about a session
func (c *Controller) GetSessionInfo(ctx context.Context, sessionName string) (*ports.TMuxSession, error) {
	args := []string{"display-message", "-t", sessionName, "-p", "#{session_name}:#{session_windows}:#{session_created}:#{session_attached}:#{session_last_attached}"}
	
	// Add socket path if configured
	if c.config.SocketPath != "" {
		args = append([]string{"-S", c.config.SocketPath}, args...)
	}

	cmd := exec.CommandContext(ctx, "tmux", args...)
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get tmux session info for %s: %w", sessionName, err)
	}

	line := strings.TrimSpace(string(output))
	parts := strings.Split(line, ":")
	if len(parts) < 5 {
		return nil, fmt.Errorf("unexpected tmux output format: %s", line)
	}

	windows, _ := strconv.Atoi(parts[1])
	
	session := &ports.TMuxSession{
		Name:     parts[0],
		Windows:  windows,
		Created:  c.formatTimestamp(parts[2]),
		Attached: parts[3] == "1",
		LastUsed: c.formatTimestamp(parts[4]),
	}

	return session, nil
}

// formatTimestamp converts tmux timestamp to readable format
func (c *Controller) formatTimestamp(timestamp string) string {
	if timestamp == "" || timestamp == "0" {
		return "N/A"
	}
	
	// tmux timestamps are in seconds since epoch
	if ts, err := strconv.ParseInt(timestamp, 10, 64); err == nil {
		return time.Unix(ts, 0).Format("2006-01-02 15:04:05")
	}
	
	return timestamp
}