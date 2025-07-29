package domain

import (
	"testing"
)

func TestHookType_IsValid(t *testing.T) {
	validTypes := []HookType{
		HookTypePreToolUse,
		HookTypePostToolUse,
		HookTypeNotification,
		HookTypeUserPromptSubmit,
		HookTypeStop,
		HookTypeSubagentStop,
		HookTypePreCompact,
	}

	for _, hookType := range validTypes {
		if !hookType.IsValid() {
			t.Errorf("Expected %s to be valid", hookType)
		}
	}

	invalidTypes := []HookType{
		"InvalidType",
		"",
		"random",
	}

	for _, hookType := range invalidTypes {
		if hookType.IsValid() {
			t.Errorf("Expected %s to be invalid", hookType)
		}
	}
}

func TestParseHookType(t *testing.T) {
	tests := []struct {
		input       string
		expected    HookType
		shouldError bool
	}{
		{"PreToolUse", HookTypePreToolUse, false},
		{"PostToolUse", HookTypePostToolUse, false},
		{"Notification", HookTypeNotification, false},
		{"  PreToolUse  ", HookTypePreToolUse, false}, // Test trimming
		{"InvalidType", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		result, err := ParseHookType(tt.input)

		if tt.shouldError {
			if err == nil {
				t.Errorf("Expected error for input %s", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("Expected %s, got %s for input %s", tt.expected, result, tt.input)
			}
		}
	}
}

func TestNewHookData(t *testing.T) {
	hookType := HookTypePreToolUse
	payload := map[string]interface{}{
		"hook_event_name": "PreToolUse",
		"session_id":      "test-session",
		"tool_name":       "Bash",
		"tool_input": map[string]interface{}{
			"command": "ls -la",
		},
	}

	hookData := NewHookData(hookType, payload)

	if hookData.Type != hookType {
		t.Errorf("Expected type %s, got %s", hookType, hookData.Type)
	}

	if hookData.GetToolName() != "Bash" {
		t.Error("Expected tool name to be extracted correctly")
	}

	if hookData.GetSessionID() != "test-session" {
		t.Error("Expected session ID to be extracted correctly")
	}
}

func TestNewHookData_NilPayload(t *testing.T) {
	hookType := HookTypeStop
	hookData := NewHookData(hookType, nil)

	if hookData.Type != hookType {
		t.Errorf("Expected type %s, got %s", hookType, hookData.Type)
	}

	if hookData.Data == nil {
		t.Error("Expected data to be initialized")
	}

	// For Stop hook, we should have structured StopHookData
	if stopData, ok := hookData.Data.(*StopHookData); ok {
		if stopData.HookEventName != "" {
			t.Error("Expected empty stop data for nil payload")
		}
	} else {
		t.Error("Expected StopHookData for Stop hook type")
	}
}

func TestNewHookDataFromRequest(t *testing.T) {
	tests := []struct {
		name        string
		req         *ClaudeCodeWebhookRequest
		expectedErr bool
		hookType    HookType
	}{
		{
			name: "PreToolUse request",
			req: &ClaudeCodeWebhookRequest{
				HookEventName:  "PreToolUse",
				SessionID:      "test-session-123",
				CWD:            "/test/path",
				TranscriptPath: "/test/transcript.md",
				ToolName:       "Bash",
				ToolInput: &ToolInput{
					Command:     "ls -la",
					Description: "List files",
				},
			},
			expectedErr: false,
			hookType:    HookTypePreToolUse,
		},
		{
			name: "Stop request",
			req: &ClaudeCodeWebhookRequest{
				HookEventName:  "Stop",
				SessionID:      "test-session-456",
				CWD:            "/test/stop",
				TranscriptPath: "/test/transcript2.md",
			},
			expectedErr: false,
			hookType:    HookTypeStop,
		},
		{
			name: "Notification request",
			req: &ClaudeCodeWebhookRequest{
				HookEventName: "Notification",
				SessionID:     "test-notification",
				Message:       "Test notification message",
			},
			expectedErr: false,
			hookType:    HookTypeNotification,
		},
		{
			name: "Invalid hook type",
			req: &ClaudeCodeWebhookRequest{
				HookEventName: "InvalidHookType",
				SessionID:     "test-invalid",
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hookData, err := NewHookDataFromRequest(tt.req)
			
			if tt.expectedErr {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
				return
			}
			
			if hookData.Type != tt.hookType {
				t.Errorf("Expected hook type %s, got %s", tt.hookType, hookData.Type)
			}
			
			if hookData.GetSessionID() != tt.req.SessionID {
				t.Errorf("Expected session ID %s, got %s", tt.req.SessionID, hookData.GetSessionID())
			}
			
			// Test specific data based on hook type
			switch tt.hookType {
			case HookTypePreToolUse:
				if hookData.GetToolName() != tt.req.ToolName {
					t.Errorf("Expected tool name %s, got %s", tt.req.ToolName, hookData.GetToolName())
				}
				if preToolData, ok := hookData.Data.(*PreToolUseHookData); ok {
					if preToolData.CWD != tt.req.CWD {
						t.Errorf("Expected CWD %s, got %s", tt.req.CWD, preToolData.CWD)
					}
					if preToolData.ToolInput == nil || preToolData.ToolInput.Command != tt.req.ToolInput.Command {
						t.Error("Expected tool input to be properly set")
					}
				} else {
					t.Error("Expected PreToolUseHookData for PreToolUse hook")
				}
			case HookTypeStop:
				if stopData, ok := hookData.Data.(*StopHookData); ok {
					if stopData.CWD != tt.req.CWD {
						t.Errorf("Expected CWD %s, got %s", tt.req.CWD, stopData.CWD)
					}
				} else {
					t.Error("Expected StopHookData for Stop hook")
				}
			case HookTypeNotification:
				if notifData, ok := hookData.Data.(*NotificationHookData); ok {
					if notifData.Message != tt.req.Message {
						t.Errorf("Expected message %s, got %s", tt.req.Message, notifData.Message)
					}
				} else {
					t.Error("Expected NotificationHookData for Notification hook")
				}
			}
		})
	}
}