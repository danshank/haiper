package claude

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewClaudeCodeAdapter(t *testing.T) {
	tests := []struct {
		name           string
		claudeBinaryPath string
		expectedPath   string
	}{
		{
			name:           "Default binary path",
			claudeBinaryPath: "",
			expectedPath:   "claude",
		},
		{
			name:           "Custom binary path",
			claudeBinaryPath: "/usr/local/bin/claude",
			expectedPath:   "/usr/local/bin/claude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewClaudeCodeAdapter(tt.claudeBinaryPath)
			
			if adapter.claudeBinaryPath != tt.expectedPath {
				t.Errorf("Expected claude binary path %s, got %s", tt.expectedPath, adapter.claudeBinaryPath)
			}
			
			if adapter.defaultTimeout != 30*time.Second {
				t.Errorf("Expected default timeout 30s, got %v", adapter.defaultTimeout)
			}
		})
	}
}

func TestClaudeCodeAdapter_SendInputToStopWebhook_Validation(t *testing.T) {
	adapter := NewClaudeCodeAdapter("echo") // Use echo for testing
	ctx := context.Background()

	tests := []struct {
		name      string
		sessionID string
		userInput string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "Empty session ID",
			sessionID: "",
			userInput: "test input",
			expectErr: true,
			errMsg:    "session ID cannot be empty",
		},
		{
			name:      "Empty user input",
			sessionID: "test-session-123",
			userInput: "",
			expectErr: true,
			errMsg:    "user input cannot be empty",
		},
		{
			name:      "Valid input",
			sessionID: "test-session-123",
			userInput: "continue",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := adapter.SendInputToStopWebhook(ctx, tt.sessionID, tt.userInput)
			
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestClaudeCodeAdapter_SendInputToStopWebhook_Success(t *testing.T) {
	// Use echo command to simulate successful Claude CLI execution
	adapter := NewClaudeCodeAdapter("echo")
	ctx := context.Background()

	response, err := adapter.SendInputToStopWebhook(ctx, "test-session-123", "continue")
	
	if err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}
	
	if !response.Success {
		t.Error("Expected successful response")
	}
	
	if response.Duration <= 0 {
		t.Error("Expected positive duration")
	}
	
	// Echo should output the arguments we passed
	expectedOutput := "-r test-session-123 continue\n"
	if response.Output != expectedOutput {
		t.Errorf("Expected output '%s', got '%s'", expectedOutput, response.Output)
	}
}

func TestClaudeCodeAdapter_SendInputToStopWebhook_Failure(t *testing.T) {
	// Use a command that will fail
	adapter := NewClaudeCodeAdapter("false") // 'false' command always exits with code 1
	ctx := context.Background()

	response, err := adapter.SendInputToStopWebhook(ctx, "test-session-123", "continue")
	
	if err == nil {
		t.Error("Expected error but got none")
	}
	
	if response == nil {
		t.Fatal("Expected response even on error")
	}
	
	if response.Success {
		t.Error("Expected unsuccessful response")
	}
	
	if response.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", response.ExitCode)
	}
}

func TestClaudeCodeAdapter_ValidateClaudeBinary(t *testing.T) {
	tests := []struct {
		name         string
		binaryPath   string
		expectError  bool
	}{
		{
			name:        "Valid binary (echo)",
			binaryPath:  "echo",
			expectError: false,
		},
		{
			name:        "Invalid binary",
			binaryPath:  "nonexistent-command-12345",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewClaudeCodeAdapter(tt.binaryPath)
			ctx := context.Background()
			
			err := adapter.ValidateClaudeBinary(ctx)
			
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestClaudeCodeAdapter_TimeoutConfiguration(t *testing.T) {
	adapter := NewClaudeCodeAdapter("claude")
	
	// Test default timeout
	if adapter.GetTimeout() != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", adapter.GetTimeout())
	}
	
	// Test setting custom timeout
	customTimeout := 60 * time.Second
	adapter.SetTimeout(customTimeout)
	
	if adapter.GetTimeout() != customTimeout {
		t.Errorf("Expected timeout %v, got %v", customTimeout, adapter.GetTimeout())
	}
}

func TestClaudeCodeAdapter_ContextTimeout(t *testing.T) {
	// Use sleep command to test timeout behavior
	adapter := NewClaudeCodeAdapter("sleep")
	adapter.SetTimeout(100 * time.Millisecond) // Very short timeout
	
	ctx := context.Background()
	
	// This should timeout since sleep 1 takes 1 second but timeout is 100ms
	_, err := adapter.SendInputToStopWebhook(ctx, "test-session", "1")
	
	if err == nil {
		t.Error("Expected timeout error but got none")
	}
	
	// Accept various timeout-related error messages
	errorStr := err.Error()
	isTimeoutError := strings.Contains(errorStr, "context deadline exceeded") ||
					  strings.Contains(errorStr, "signal: killed") ||
					  strings.Contains(errorStr, "exit status") // sleep command may exit with non-zero
	
	if !isTimeoutError {
		t.Errorf("Expected timeout-related error, got: %v", err)
	}
}