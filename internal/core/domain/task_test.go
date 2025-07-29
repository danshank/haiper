package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewTask(t *testing.T) {
	payload := map[string]interface{}{
		"hook_event_name": "PreToolUse",
		"session_id":      "test-session-123",
		"tool_name":       "Bash",
		"tool_input": map[string]interface{}{
			"command": "ls -la",
		},
	}
	hookData := NewHookData(HookTypePreToolUse, payload)

	task := NewTask(hookData)

	if task.ID == uuid.Nil {
		t.Error("Expected task ID to be generated")
	}

	if task.HookType != HookTypePreToolUse {
		t.Errorf("Expected hook type %s, got %s", HookTypePreToolUse, task.HookType)
	}

	if task.HookData == nil {
		t.Error("Expected hook data to be set")
	}

	if task.HookData.GetSessionID() != "test-session-123" {
		t.Errorf("Expected session ID test-session-123, got %s", task.HookData.GetSessionID())
	}

	if task.Status != TaskStatusPending {
		t.Errorf("Expected status %s, got %s", TaskStatusPending, task.Status)
	}

	if task.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if task.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

func TestTask_UpdateStatus(t *testing.T) {
	hookData := NewHookData(HookTypePreToolUse, map[string]interface{}{})
	task := NewTask(hookData)
	originalUpdatedAt := task.UpdatedAt

	// Wait a tiny bit to ensure timestamp changes
	time.Sleep(time.Millisecond)

	task.UpdateStatus(TaskStatusApproved)

	if task.Status != TaskStatusApproved {
		t.Errorf("Expected status %s, got %s", TaskStatusApproved, task.Status)
	}

	if !task.UpdatedAt.After(originalUpdatedAt) {
		t.Error("Expected UpdatedAt to be updated")
	}
}

func TestTask_TakeAction(t *testing.T) {
	hookData := NewHookData(HookTypePreToolUse, map[string]interface{}{})
	task := NewTask(hookData)
	responseData := map[string]interface{}{
		"reason": "Approved by user",
	}

	task.TakeAction(ActionTypeApprove, responseData)

	if task.ActionTaken == nil {
		t.Fatal("Expected ActionTaken to be set")
	}

	if *task.ActionTaken != ActionTypeApprove {
		t.Errorf("Expected action %s, got %s", ActionTypeApprove, *task.ActionTaken)
	}

	if task.Status != TaskStatusApproved {
		t.Errorf("Expected status %s, got %s", TaskStatusApproved, task.Status)
	}

	if task.ResponseData["reason"] != "Approved by user" {
		t.Error("Expected response data to be set")
	}
}

func TestTask_IsActionable(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected bool
	}{
		{TaskStatusPending, true},
		{TaskStatusApproved, false},
		{TaskStatusRejected, false},
		{TaskStatusCompleted, false},
		{TaskStatusFailed, false},
	}

	for _, tt := range tests {
		hookData := NewHookData(HookTypePreToolUse, map[string]interface{}{})
		task := NewTask(hookData)
		task.Status = tt.status

		if task.IsActionable() != tt.expected {
			t.Errorf("Expected IsActionable() to return %t for status %s", tt.expected, tt.status)
		}
	}
}

func TestTask_RequiresUserInput(t *testing.T) {
	tests := []struct {
		hookType HookType
		expected bool
	}{
		{HookTypePreToolUse, true},
		{HookTypeUserPromptSubmit, true},
		{HookTypePostToolUse, false},
		{HookTypeNotification, false}, // Now non-blocking
		{HookTypeStop, false},         // Now non-blocking
		{HookTypeSubagentStop, false},
		{HookTypePreCompact, false},
	}

	for _, tt := range tests {
		hookData := NewHookData(tt.hookType, map[string]interface{}{})
		task := NewTask(hookData)

		if task.RequiresUserInput() != tt.expected {
			t.Errorf("Expected RequiresUserInput() to return %t for hook type %s", tt.expected, tt.hookType)
		}
	}
}