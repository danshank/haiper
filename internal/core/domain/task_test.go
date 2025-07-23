package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewTask(t *testing.T) {
	hookType := HookTypePreToolUse
	taskData := json.RawMessage(`{"tool": "Bash", "command": "ls -la"}`)

	task := NewTask(hookType, taskData)

	if task.ID == uuid.Nil {
		t.Error("Expected task ID to be generated")
	}

	if task.HookType != hookType {
		t.Errorf("Expected hook type %s, got %s", hookType, task.HookType)
	}

	if string(task.TaskData) != string(taskData) {
		t.Errorf("Expected task data %s, got %s", taskData, task.TaskData)
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
	task := NewTask(HookTypePreToolUse, json.RawMessage(`{}`))
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
	task := NewTask(HookTypePreToolUse, json.RawMessage(`{}`))
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
		task := NewTask(HookTypePreToolUse, json.RawMessage(`{}`))
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
		{HookTypeNotification, true},
		{HookTypeUserPromptSubmit, true},
		{HookTypePostToolUse, false},
		{HookTypeStop, false},
		{HookTypeSubagentStop, false},
		{HookTypePreCompact, false},
	}

	for _, tt := range tests {
		task := NewTask(tt.hookType, json.RawMessage(`{}`))

		if task.RequiresUserInput() != tt.expected {
			t.Errorf("Expected RequiresUserInput() to return %t for hook type %s", tt.expected, tt.hookType)
		}
	}
}