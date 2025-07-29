package claude

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// ClaudeCodeAdapter handles interaction with Claude Code CLI for specific webhook types
type ClaudeCodeAdapter struct {
	claudeBinaryPath string
	defaultTimeout   time.Duration
}

// ClaudeSession represents a Claude Code session that can receive input
type ClaudeSession struct {
	SessionID string
	HookType  string
}

// ClaudeResponse represents the result of sending input to Claude Code
type ClaudeResponse struct {
	Success   bool
	Output    string
	Error     string
	ExitCode  int
	Duration  time.Duration
}

// NewClaudeCodeAdapter creates a new Claude Code CLI adapter
func NewClaudeCodeAdapter(claudeBinaryPath string) *ClaudeCodeAdapter {
	if claudeBinaryPath == "" {
		claudeBinaryPath = "claude" // Assume claude is in PATH
	}
	
	return &ClaudeCodeAdapter{
		claudeBinaryPath: claudeBinaryPath,
		defaultTimeout:   30 * time.Second,
	}
}

// SendInputToStopWebhook sends user-defined input to a Claude Code session for Stop webhook
func (c *ClaudeCodeAdapter) SendInputToStopWebhook(ctx context.Context, sessionID, userInput string) (*ClaudeResponse, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session ID cannot be empty")
	}
	
	if userInput == "" {
		return nil, fmt.Errorf("user input cannot be empty")
	}

	// Create context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
	defer cancel()

	// Build the claude command: claude -r "<session-id>" "user-input"
	cmd := exec.CommandContext(cmdCtx, c.claudeBinaryPath, "-r", sessionID, userInput)
	
	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	response := &ClaudeResponse{
		Success:  err == nil,
		Output:   string(output),
		Duration: duration,
	}

	if err != nil {
		response.Error = err.Error()
		if exitError, ok := err.(*exec.ExitError); ok {
			response.ExitCode = exitError.ExitCode()
		}
		return response, fmt.Errorf("failed to send input to Claude Code session %s: %w", sessionID, err)
	}

	return response, nil
}

// ValidateClaudeBinary checks if the Claude Code CLI is available and working
func (c *ClaudeCodeAdapter) ValidateClaudeBinary(ctx context.Context) error {
	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	// Try to run claude --help to verify it's installed and accessible
	cmd := exec.CommandContext(cmdCtx, c.claudeBinaryPath, "--help")
	_, err := cmd.CombinedOutput()
	
	if err != nil {
		return fmt.Errorf("Claude Code CLI not found or not working at path '%s': %w", c.claudeBinaryPath, err)
	}
	
	return nil
}

// SetTimeout configures the timeout for Claude Code CLI operations
func (c *ClaudeCodeAdapter) SetTimeout(timeout time.Duration) {
	c.defaultTimeout = timeout
}

// GetTimeout returns the current timeout setting
func (c *ClaudeCodeAdapter) GetTimeout() time.Duration {
	return c.defaultTimeout
}