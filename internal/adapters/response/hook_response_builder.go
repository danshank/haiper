package response

import (
	"time"

	"github.com/dan/claude-control/internal/core/domain"
	"github.com/dan/claude-control/internal/core/ports"
)

// HookResponseBuilder implements the HookResponseBuilder port
type HookResponseBuilder struct{}

// NewHookResponseBuilder creates a new hook response builder
func NewHookResponseBuilder() ports.HookResponseBuilder {
	return &HookResponseBuilder{}
}

// BuildBlockingResponse creates a response that blocks Claude Code execution
func (b *HookResponseBuilder) BuildBlockingResponse(taskID, reason string) *domain.HookResponse {
	return domain.NewBlockingResponse(taskID, reason)
}

// BuildApprovedResponse creates a response that allows Claude Code to continue
func (b *HookResponseBuilder) BuildApprovedResponse(taskID string) *domain.HookResponse {
	return domain.NewApprovedResponse(taskID)
}

// BuildRejectedResponse creates a response that blocks Claude Code with user rejection
func (b *HookResponseBuilder) BuildRejectedResponse(taskID, reason string) *domain.HookResponse {
	return domain.NewRejectedResponse(taskID, reason)
}

// BuildTimeoutResponse creates a response for when user decision times out
func (b *HookResponseBuilder) BuildTimeoutResponse(taskID string, timeout time.Duration) *domain.HookResponse {
	return domain.NewTimeoutResponse(taskID, timeout)
}

// BuildContinueResponse creates a non-blocking response that allows continuation
func (b *HookResponseBuilder) BuildContinueResponse() *domain.HookResponse {
	return domain.NewContinueResponse()
}

// BuildSuppressedResponse creates a non-blocking response with suppressed output
func (b *HookResponseBuilder) BuildSuppressedResponse() *domain.HookResponse {
	return domain.NewSuppressedResponse()
}

// BuildResponseFromDecision creates appropriate response based on user decision
func (b *HookResponseBuilder) BuildResponseFromDecision(taskID string, decision domain.ActionType) *domain.HookResponse {
	switch decision {
	case domain.ActionTypeApprove:
		return b.BuildApprovedResponse(taskID)
	case domain.ActionTypeReject:
		return b.BuildRejectedResponse(taskID, "User rejected this action")
	case domain.ActionTypeCancel:
		return b.BuildRejectedResponse(taskID, "User cancelled this action")
	default:
		// For any other action, default to approved
		return b.BuildApprovedResponse(taskID)
	}
}