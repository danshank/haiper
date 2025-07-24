package ports

import (
	"context"
	"time"

	"github.com/dan/claude-control/internal/core/domain"
)

// HookResponseBuilder creates Claude Code compliant JSON responses
type HookResponseBuilder interface {
	// BuildBlockingResponse creates a response that blocks Claude Code execution
	// Used when user approval is required
	BuildBlockingResponse(taskID, reason string) *domain.HookResponse

	// BuildApprovedResponse creates a response that allows Claude Code to continue
	// Used when user approves the action
	BuildApprovedResponse(taskID string) *domain.HookResponse

	// BuildRejectedResponse creates a response that blocks Claude Code with user rejection
	// Used when user rejects the action
	BuildRejectedResponse(taskID, reason string) *domain.HookResponse

	// BuildTimeoutResponse creates a response for when user decision times out
	// Used when no user decision is received within the timeout period
	BuildTimeoutResponse(taskID string, timeout time.Duration) *domain.HookResponse

	// BuildContinueResponse creates a non-blocking response that allows continuation
	// Used for hooks that don't require user approval
	BuildContinueResponse() *domain.HookResponse

	// BuildSuppressedResponse creates a non-blocking response with suppressed output
	// Used for hooks that should continue but hide their output
	BuildSuppressedResponse() *domain.HookResponse

	// BuildResponseFromDecision creates appropriate response based on user decision
	BuildResponseFromDecision(taskID string, decision domain.ActionType) *domain.HookResponse
}

// HookResponseSender sends hook responses back to Claude Code
type HookResponseSender interface {
	// SendResponse sends a JSON response to Claude Code
	// The response is written to the HTTP response writer
	SendResponse(ctx context.Context, response *domain.HookResponse) error

	// SendBlockingResponse sends a blocking response and waits for user decision
	// Returns the final response to send to Claude Code
	SendBlockingResponse(ctx context.Context, taskID string, timeout time.Duration) (*domain.HookResponse, error)
}

// HookResponseValidator validates hook responses comply with Claude Code spec
type HookResponseValidator interface {
	// ValidateResponse checks if response format is valid for Claude Code
	ValidateResponse(response *domain.HookResponse) error

	// ValidateJSON checks if JSON bytes are valid Claude Code response
	ValidateJSON(jsonBytes []byte) error
}