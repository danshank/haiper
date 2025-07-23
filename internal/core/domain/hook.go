package domain

import (
	"fmt"
	"strings"
)

// HookType represents the official Claude Code hook event types
// Reference: https://docs.anthropic.com/en/docs/claude-code/hooks
type HookType string

const (
	// HookTypePreToolUse runs after Claude creates tool parameters and before processing the tool call
	HookTypePreToolUse HookType = "PreToolUse"
	
	// HookTypePostToolUse runs immediately after a tool completes successfully
	HookTypePostToolUse HookType = "PostToolUse"
	
	// HookTypeNotification runs when Claude Code sends notifications
	// (when Claude needs permission to use a tool or prompt idle for 60+ seconds)
	HookTypeNotification HookType = "Notification"
	
	// HookTypeUserPromptSubmit runs when the user submits a prompt, before Claude processes it
	HookTypeUserPromptSubmit HookType = "UserPromptSubmit"
	
	// HookTypeStop runs when the main Claude Code agent has finished responding
	HookTypeStop HookType = "Stop"
	
	// HookTypeSubagentStop runs when a Claude Code subagent (Task tool call) has finished responding
	HookTypeSubagentStop HookType = "SubagentStop"
	
	// HookTypePreCompact runs before Claude Code runs a compact operation
	HookTypePreCompact HookType = "PreCompact"
)

func (h HookType) String() string {
	return string(h)
}

func (h HookType) IsValid() bool {
	switch h {
	case HookTypePreToolUse, HookTypePostToolUse, HookTypeNotification, 
		 HookTypeUserPromptSubmit, HookTypeStop, HookTypeSubagentStop, HookTypePreCompact:
		return true
	default:
		return false
	}
}

func ParseHookType(s string) (HookType, error) {
	hookType := HookType(strings.TrimSpace(s))
	if !hookType.IsValid() {
		return "", fmt.Errorf("invalid hook type: %s", s)
	}
	return hookType, nil
}

// HookData represents the data payload from Claude Code hooks
type HookData struct {
	Type    HookType               `json:"type"`
	Tool    string                 `json:"tool,omitempty"`    // For PreToolUse/PostToolUse
	Matcher string                 `json:"matcher,omitempty"` // For PreCompact (manual/auto)
	Payload map[string]interface{} `json:"payload"`
}

func NewHookData(hookType HookType, payload map[string]interface{}) *HookData {
	if payload == nil {
		payload = make(map[string]interface{})
	}
	return &HookData{
		Type:    hookType,
		Payload: payload,
	}
}