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
	HookEventName  string                 `json:"hook_event_name"`
	SessionID      string                 `json:"session_id"`
	CWD            string                 `json:"cwd,omitempty"`
	TranscriptPath string                 `json:"transcript_path,omitempty"`
	ToolName       string                 `json:"tool_name,omitempty"`
	ToolInput      *ToolInput             `json:"tool_input,omitempty"`
	ToolResponse   *ToolResponse          `json:"tool_response,omitempty"`
	Message        string                 `json:"message,omitempty"`
	UserPrompt     string                 `json:"user_prompt,omitempty"`
	SubagentID     string                 `json:"subagent_id,omitempty"`
	Matcher        string                 `json:"matcher,omitempty"`
}

// ToolInput represents tool input parameters from Claude Code
type ToolInput struct {
	Command     string `json:"command,omitempty"`
	Description string `json:"description,omitempty"`
}

// ToolResponse represents tool execution results from Claude Code
type ToolResponse struct {
	Interrupted bool   `json:"interrupted"`
	IsImage     bool   `json:"isImage"`
	Stderr      string `json:"stderr"`
	Stdout      string `json:"stdout"`
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
	// Stop webhooks typically contain minimal data, mainly session information
}

// SubagentStopHookData represents data from SubagentStop webhooks
type SubagentStopHookData struct {
	BaseHookData
	SubagentID string `json:"subagent_id,omitempty"`
}

// PreCompactHookData represents data from PreCompact webhooks
type PreCompactHookData struct {
	BaseHookData
	Matcher string `json:"matcher,omitempty"` // "manual" or "auto"
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

// NewHookData creates structured hook data based on the hook type (kept for backward compatibility)
func NewHookData(hookType HookType, rawPayload map[string]interface{}) *HookData {
	var structuredData interface{}
	
	switch hookType {
	case HookTypePreToolUse:
		data := &PreToolUseHookData{}
		populateFromMap(rawPayload, data)
		structuredData = data
	case HookTypePostToolUse:
		data := &PostToolUseHookData{}
		populateFromMap(rawPayload, data)
		structuredData = data
	case HookTypeNotification:
		data := &NotificationHookData{}
		populateFromMap(rawPayload, data)
		structuredData = data
	case HookTypeUserPromptSubmit:
		data := &UserPromptSubmitHookData{}
		populateFromMap(rawPayload, data)
		structuredData = data
	case HookTypeStop:
		data := &StopHookData{}
		populateFromMap(rawPayload, data)
		structuredData = data
	case HookTypeSubagentStop:
		data := &SubagentStopHookData{}
		populateFromMap(rawPayload, data)
		structuredData = data
	case HookTypePreCompact:
		data := &PreCompactHookData{}
		populateFromMap(rawPayload, data)
		structuredData = data
	default:
		// Fallback to generic payload for unknown hook types
		structuredData = rawPayload
	}
	
	return &HookData{
		Type: hookType,
		Data: structuredData,
	}
}

// GetSessionID extracts the session ID from any hook data type
func (h *HookData) GetSessionID() string {
	switch data := h.Data.(type) {
	case *PreToolUseHookData:
		return data.SessionID
	case *PostToolUseHookData:
		return data.SessionID
	case *NotificationHookData:
		return data.SessionID
	case *UserPromptSubmitHookData:
		return data.SessionID
	case *StopHookData:
		return data.SessionID
	case *SubagentStopHookData:
		return data.SessionID
	case *PreCompactHookData:
		return data.SessionID
	default:
		// Try to extract from map for fallback cases
		if payload, ok := data.(map[string]interface{}); ok {
			if sessionID, exists := payload["session_id"]; exists {
				if sessionIDStr, ok := sessionID.(string); ok {
					return sessionIDStr
				}
			}
		}
		return ""
	}
}

// GetToolName extracts the tool name from hook data (for tool-related hooks)
func (h *HookData) GetToolName() string {
	switch data := h.Data.(type) {
	case *PreToolUseHookData:
		return data.ToolName
	case *PostToolUseHookData:
		return data.ToolName
	default:
		return ""
	}
}

// populateFromMap populates a struct from a map using reflection-like field matching
func populateFromMap(source map[string]interface{}, target interface{}) {
	// This is a simplified version - in production, you'd use reflection or a library like mapstructure
	// For now, we'll implement the specific cases we need
	
	switch t := target.(type) {
	case *PreToolUseHookData:
		if v, ok := source["hook_event_name"].(string); ok {
			t.HookEventName = v
		}
		if v, ok := source["session_id"].(string); ok {
			t.SessionID = v
		}
		if v, ok := source["cwd"].(string); ok {
			t.CWD = v
		}
		if v, ok := source["transcript_path"].(string); ok {
			t.TranscriptPath = v
		}
		if v, ok := source["tool_name"].(string); ok {
			t.ToolName = v
		}
		if toolInput, ok := source["tool_input"].(map[string]interface{}); ok {
			ti := &ToolInput{}
			if cmd, ok := toolInput["command"].(string); ok {
				ti.Command = cmd
			}
			if desc, ok := toolInput["description"].(string); ok {
				ti.Description = desc
			}
			t.ToolInput = ti
		}
	case *PostToolUseHookData:
		if v, ok := source["hook_event_name"].(string); ok {
			t.HookEventName = v
		}
		if v, ok := source["session_id"].(string); ok {
			t.SessionID = v
		}
		if v, ok := source["cwd"].(string); ok {
			t.CWD = v
		}
		if v, ok := source["transcript_path"].(string); ok {
			t.TranscriptPath = v
		}
		if v, ok := source["tool_name"].(string); ok {
			t.ToolName = v
		}
		if toolInput, ok := source["tool_input"].(map[string]interface{}); ok {
			ti := &ToolInput{}
			if cmd, ok := toolInput["command"].(string); ok {
				ti.Command = cmd
			}
			if desc, ok := toolInput["description"].(string); ok {
				ti.Description = desc
			}
			t.ToolInput = ti
		}
		if toolResponse, ok := source["tool_response"].(map[string]interface{}); ok {
			tr := &ToolResponse{}
			if interrupted, ok := toolResponse["interrupted"].(bool); ok {
				tr.Interrupted = interrupted
			}
			if isImage, ok := toolResponse["isImage"].(bool); ok {
				tr.IsImage = isImage
			}
			if stderr, ok := toolResponse["stderr"].(string); ok {
				tr.Stderr = stderr
			}
			if stdout, ok := toolResponse["stdout"].(string); ok {
				tr.Stdout = stdout
			}
			t.ToolResponse = tr
		}
	case *NotificationHookData:
		if v, ok := source["hook_event_name"].(string); ok {
			t.HookEventName = v
		}
		if v, ok := source["session_id"].(string); ok {
			t.SessionID = v
		}
		if v, ok := source["cwd"].(string); ok {
			t.CWD = v
		}
		if v, ok := source["transcript_path"].(string); ok {
			t.TranscriptPath = v
		}
		if v, ok := source["message"].(string); ok {
			t.Message = v
		}
	case *UserPromptSubmitHookData:
		if v, ok := source["hook_event_name"].(string); ok {
			t.HookEventName = v
		}
		if v, ok := source["session_id"].(string); ok {
			t.SessionID = v
		}
		if v, ok := source["cwd"].(string); ok {
			t.CWD = v
		}
		if v, ok := source["transcript_path"].(string); ok {
			t.TranscriptPath = v
		}
		if v, ok := source["user_prompt"].(string); ok {
			t.UserPrompt = v
		}
	case *StopHookData:
		if v, ok := source["hook_event_name"].(string); ok {
			t.HookEventName = v
		}
		if v, ok := source["session_id"].(string); ok {
			t.SessionID = v
		}
		if v, ok := source["cwd"].(string); ok {
			t.CWD = v
		}
		if v, ok := source["transcript_path"].(string); ok {
			t.TranscriptPath = v
		}
	case *SubagentStopHookData:
		if v, ok := source["hook_event_name"].(string); ok {
			t.HookEventName = v
		}
		if v, ok := source["session_id"].(string); ok {
			t.SessionID = v
		}
		if v, ok := source["cwd"].(string); ok {
			t.CWD = v
		}
		if v, ok := source["transcript_path"].(string); ok {
			t.TranscriptPath = v
		}
		if v, ok := source["subagent_id"].(string); ok {
			t.SubagentID = v
		}
	case *PreCompactHookData:
		if v, ok := source["hook_event_name"].(string); ok {
			t.HookEventName = v
		}
		if v, ok := source["session_id"].(string); ok {
			t.SessionID = v
		}
		if v, ok := source["cwd"].(string); ok {
			t.CWD = v
		}
		if v, ok := source["transcript_path"].(string); ok {
			t.TranscriptPath = v
		}
		if v, ok := source["matcher"].(string); ok {
			t.Matcher = v
		}
	}
}