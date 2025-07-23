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
		"tool":    "Bash",
		"command": "ls -la",
	}

	hookData := NewHookData(hookType, payload)

	if hookData.Type != hookType {
		t.Errorf("Expected type %s, got %s", hookType, hookData.Type)
	}

	if hookData.Payload["tool"] != "Bash" {
		t.Error("Expected payload to be set correctly")
	}
}

func TestNewHookData_NilPayload(t *testing.T) {
	hookType := HookTypeStop
	hookData := NewHookData(hookType, nil)

	if hookData.Type != hookType {
		t.Errorf("Expected type %s, got %s", hookType, hookData.Type)
	}

	if hookData.Payload == nil {
		t.Error("Expected payload to be initialized as empty map")
	}

	if len(hookData.Payload) != 0 {
		t.Error("Expected payload to be empty")
	}
}