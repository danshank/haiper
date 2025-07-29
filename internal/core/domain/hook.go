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

// ClaudeCodeWebhookRequest represents the common structure of Claude Code webhook requests
type ClaudeCodeWebhookRequest struct {
	HookEventName  string        `json:"hook_event_name"`
	SessionID      string        `json:"session_id"`
	CWD            string        `json:"cwd,omitempty"`
	TranscriptPath string        `json:"transcript_path,omitempty"`
	ToolName       string        `json:"tool_name,omitempty"`
	ToolInput      *ToolInput    `json:"tool_input,omitempty"`
	ToolResponse   *ToolResponse `json:"tool_response,omitempty"`
	Message        string        `json:"message,omitempty"`
	UserPrompt     string        `json:"user_prompt,omitempty"`
	SubagentID     string        `json:"subagent_id,omitempty"`
	Matcher        string        `json:"matcher,omitempty"`
}

// ToolInput represents tool input parameters from Claude Code
type ToolInput struct {
	Command     string `json:"command,omitempty"`
	Description string `json:"description,omitempty"`
}

// ToolResponse represents tool execution results from Claude Code
type ToolResponse struct {
	Interrupted bool   `json:"interrupted,omitempty"`
	Stderr      string `json:"stderr,omitempty"`
	Stdout      string `json:"stdout,omitempty"`
	Success     bool   `json:"success,omitempty"`
}

// BaseHookData contains common fields present in all Claude Code webhooks
type BaseHookData struct {
	HookEventName  string `json:"hook_event_name"`
	SessionID      string `json:"session_id"`
	CWD            string `json:"cwd,omitempty"`
	TranscriptPath string `json:"transcript_path,omitempty"`
}

// PreToolUseHookData represents data from PreToolUse webhooks
type PreToolUseHookData struct {
	BaseHookData
	ToolName  string     `json:"tool_name"`
	ToolInput *ToolInput `json:"tool_input,omitempty"`
}

// PostToolUseHookData represents data from PostToolUse webhooks
type PostToolUseHookData struct {
	BaseHookData
	ToolName     string        `json:"tool_name"`
	ToolInput    *ToolInput    `json:"tool_input,omitempty"`
	ToolResponse *ToolResponse `json:"tool_response,omitempty"`
}

// NotificationHookData represents data from Notification webhooks
type NotificationHookData struct {
	BaseHookData
	Message string `json:"message,omitempty"`
}

// UserPromptSubmitHookData represents data from UserPromptSubmit webhooks
type UserPromptSubmitHookData struct {
	BaseHookData
	UserPrompt string `json:"user_prompt,omitempty"`
}

// StopHookData represents data from Stop webhooks
type StopHookData struct {
	BaseHookData
	StopHookActive bool `json:"stop_hook_active"`
	// Stop webhooks typically contain minimal data, mainly session information
}

// SubagentStopHookData represents data from SubagentStop webhooks
type SubagentStopHookData struct {
	BaseHookData
	StopHookActive bool   `json:"stop_hook_active"`
	SubagentID     string `json:"subagent_id,omitempty"`
}

// PreCompactHookData represents data from PreCompact webhooks
type PreCompactHookData struct {
	BaseHookData
	Trigger            string `json:"trigger,omitempty"` // "manual" or "auto"
	CustomInstructions string `json:"custom_instructions,omitempty"`
}

// HookData represents the unified hook data structure
type HookData struct {
	Type HookType    `json:"type"`
	Data interface{} `json:"data"`
}

// NewHookDataFromRequest creates structured hook data from a webhook request
func NewHookDataFromRequest(req *ClaudeCodeWebhookRequest) (*HookData, error) {
	// Parse hook type from request
	hookType, err := ParseHookType(req.HookEventName)
	if err != nil {
		return nil, fmt.Errorf("invalid hook type: %w", err)
	}

	var structuredData interface{}

	switch hookType {
	case HookTypePreToolUse:
		structuredData = &PreToolUseHookData{
			BaseHookData: BaseHookData{
				HookEventName:  req.HookEventName,
				SessionID:      req.SessionID,
				CWD:            req.CWD,
				TranscriptPath: req.TranscriptPath,
			},
			ToolName:  req.ToolName,
			ToolInput: req.ToolInput,
		}
	case HookTypePostToolUse:
		structuredData = &PostToolUseHookData{
			BaseHookData: BaseHookData{
				HookEventName:  req.HookEventName,
				SessionID:      req.SessionID,
				CWD:            req.CWD,
				TranscriptPath: req.TranscriptPath,
			},
			ToolName:     req.ToolName,
			ToolInput:    req.ToolInput,
			ToolResponse: req.ToolResponse,
		}
	case HookTypeNotification:
		structuredData = &NotificationHookData{
			BaseHookData: BaseHookData{
				HookEventName:  req.HookEventName,
				SessionID:      req.SessionID,
				CWD:            req.CWD,
				TranscriptPath: req.TranscriptPath,
			},
			Message: req.Message,
		}
	case HookTypeUserPromptSubmit:
		structuredData = &UserPromptSubmitHookData{
			BaseHookData: BaseHookData{
				HookEventName:  req.HookEventName,
				SessionID:      req.SessionID,
				CWD:            req.CWD,
				TranscriptPath: req.TranscriptPath,
			},
			UserPrompt: req.UserPrompt,
		}
	case HookTypeStop:
		structuredData = &StopHookData{
			BaseHookData: BaseHookData{
				HookEventName:  req.HookEventName,
				SessionID:      req.SessionID,
				CWD:            req.CWD,
				TranscriptPath: req.TranscriptPath,
			},
		}
	case HookTypeSubagentStop:
		structuredData = &SubagentStopHookData{
			BaseHookData: BaseHookData{
				HookEventName:  req.HookEventName,
				SessionID:      req.SessionID,
				CWD:            req.CWD,
				TranscriptPath: req.TranscriptPath,
			},
			SubagentID: req.SubagentID,
		}
	case HookTypePreCompact:
		structuredData = &PreCompactHookData{
			BaseHookData: BaseHookData{
				HookEventName:  req.HookEventName,
				SessionID:      req.SessionID,
				CWD:            req.CWD,
				TranscriptPath: req.TranscriptPath,
			},
			Matcher: req.Matcher,
		}
	default:
		return nil, fmt.Errorf("unsupported hook type: %s", hookType)
	}

	return &HookData{
		Type: hookType,
		Data: structuredData,
	}, nil
}
