package domain

import (
	"encoding/json"
	"time"
)

// HookResponse represents the JSON response format expected by Claude Code hooks
// Based on https://docs.anthropic.com/en/docs/claude-code/hooks#hook-output
type HookResponse struct {
	// Continue determines if Claude Code should proceed with the action
	// false = block/stop, true = allow/continue
	Continue bool `json:"continue"`

	// StopReason provides a message to Claude Code when Continue is false
	// This message is shown to the user explaining why the action was blocked
	StopReason string `json:"stopReason,omitempty"`

	// SuppressOutput controls whether stdout from the hook is hidden
	// true = hide output, false = show output (default)
	SuppressOutput bool `json:"suppressOutput,omitempty"`

	// Metadata for internal tracking
	TaskID    string    `json:"-"` // Internal - not sent to Claude Code
	Decision  ActionType `json:"-"` // Internal - tracks user decision
	CreatedAt time.Time `json:"-"` // Internal - when response was created
}

// HookResponseType represents different types of hook responses
type HookResponseType string

const (
	// Blocking responses - require user decision
	HookResponseBlocking   HookResponseType = "blocking"
	HookResponseApproved   HookResponseType = "approved"
	HookResponseRejected   HookResponseType = "rejected"
	HookResponseTimeout    HookResponseType = "timeout"
	
	// Non-blocking responses - automatic decisions
	HookResponseContinue   HookResponseType = "continue"
	HookResponseSuppressed HookResponseType = "suppressed"
)

// NewBlockingResponse creates a response that blocks Claude Code execution
func NewBlockingResponse(taskID, reason string) *HookResponse {
	return &HookResponse{
		Continue:   false,
		StopReason: reason,
		TaskID:     taskID,
		CreatedAt:  time.Now(),
	}
}

// NewApprovedResponse creates a response that allows Claude Code to continue
func NewApprovedResponse(taskID string) *HookResponse {
	return &HookResponse{
		Continue:  true,
		TaskID:    taskID,
		Decision:  ActionTypeApprove,
		CreatedAt: time.Now(),
	}
}

// NewRejectedResponse creates a response that blocks Claude Code with user rejection
func NewRejectedResponse(taskID, reason string) *HookResponse {
	return &HookResponse{
		Continue:   false,
		StopReason: reason,
		TaskID:     taskID,
		Decision:   ActionTypeReject,
		CreatedAt:  time.Now(),
	}
}

// NewTimeoutResponse creates a response for when user decision times out
func NewTimeoutResponse(taskID string, timeout time.Duration) *HookResponse {
	return &HookResponse{
		Continue:   false,
		StopReason: "User decision timeout after " + timeout.String(),
		TaskID:     taskID,
		CreatedAt:  time.Now(),
	}
}

// NewContinueResponse creates a non-blocking response that allows continuation
func NewContinueResponse() *HookResponse {
	return &HookResponse{
		Continue:  true,
		CreatedAt: time.Now(),
	}
}

// NewSuppressedResponse creates a non-blocking response with suppressed output
func NewSuppressedResponse() *HookResponse {
	return &HookResponse{
		Continue:       true,
		SuppressOutput: true,
		CreatedAt:      time.Now(),
	}
}

// ToJSON converts the hook response to JSON bytes for Claude Code
func (hr *HookResponse) ToJSON() ([]byte, error) {
	return json.Marshal(hr)
}

// String returns a human-readable description of the response
func (hr *HookResponse) String() string {
	if hr.Continue {
		if hr.SuppressOutput {
			return "Continue (output suppressed)"
		}
		return "Continue"
	}
	
	if hr.StopReason != "" {
		return "Block: " + hr.StopReason
	}
	
	return "Block"
}

// IsBlocking returns true if this response will block Claude Code execution
func (hr *HookResponse) IsBlocking() bool {
	return !hr.Continue
}

// GetResponseType categorizes the type of hook response
func (hr *HookResponse) GetResponseType() HookResponseType {
	if hr.Continue {
		if hr.SuppressOutput {
			return HookResponseSuppressed
		}
		if hr.Decision == ActionTypeApprove {
			return HookResponseApproved
		}
		return HookResponseContinue
	}
	
	// Not continuing - determine why
	if hr.Decision == ActionTypeReject {
		return HookResponseRejected
	}
	
	if hr.StopReason != "" && hr.StopReason[:7] == "timeout" {
		return HookResponseTimeout
	}
	
	return HookResponseBlocking
}