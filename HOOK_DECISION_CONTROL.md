# Hook Decision Control Implementation

## Current Problem

Our webhook handlers currently only collect hook data but don't provide decision control back to Claude Code. This means:

1. **PreToolUse hooks can't block tool execution**
2. **No real-time approval/rejection flow**
3. **Claude Code continues regardless of user decision**

## Required Implementation

### 1. Synchronous Hook Response

For hooks that need decision control (especially PreToolUse), we need to:

```go
func (h *WebhookHandler) handlePreToolUse(w http.ResponseWriter, r *http.Request) {
    // Parse hook data
    var payload map[string]interface{}
    json.NewDecoder(r.Body).Decode(&payload)
    
    // Check if this tool requires approval
    if h.requiresApproval(payload) {
        // Create task and wait for user decision
        task := h.createTaskAndWaitForDecision(r.Context(), payload)
        
        // Respond based on user decision
        if task.Status == domain.TaskStatusApproved {
            h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
                "continue": true,
            })
        } else {
            h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
                "continue": false,
                "stopReason": "User rejected tool execution",
            })
        }
    } else {
        // Auto-approve non-critical tools
        h.respondWithJSON(w, http.StatusOK, map[string]interface{}{
            "continue": true,
        })
    }
}
```

### 2. Real-Time Decision Waiting

We need a mechanism to wait for user decisions:

```go
func (s *TaskService) CreateTaskAndWaitForDecision(ctx context.Context, hookData *domain.HookData, timeout time.Duration) (*domain.Task, error) {
    // Create task
    task := domain.NewTask(hookData.Type, taskData)
    s.taskRepo.Create(ctx, task)
    
    // Send notification
    s.sendNotification(ctx, task)
    
    // Wait for user decision with timeout
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    timeoutTimer := time.NewTimer(timeout)
    defer timeoutTimer.Stop()
    
    for {
        select {
        case <-timeoutTimer.C:
            // Timeout - auto-approve or reject based on policy
            return task, fmt.Errorf("timeout waiting for user decision")
            
        case <-ticker.C:
            // Check if user has made a decision
            updatedTask, err := s.taskRepo.GetByID(ctx, task.ID)
            if err != nil {
                continue
            }
            
            if !updatedTask.IsActionable() {
                return updatedTask, nil
            }
        }
    }
}
```

### 3. Hook Configuration Updates

Update the hook commands to handle responses:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "response=$(curl -s -X POST http://localhost:10291/webhook/pre-tool-use -H 'Content-Type: application/json' -d '{\"hook_type\": \"PreToolUse\", \"session_id\": \"'\"$session_id\"'\", \"tool\": \"'\"$tool_name\"'\", \"data\": '\"$tool_input\"'}'); echo \"$response\"; if echo \"$response\" | jq -e '.continue == false' > /dev/null; then exit 2; fi",
            "timeout": 300
          }
        ]
      }
    ]
  }
}
```

## Implementation Options

### Option 1: Synchronous Blocking (Real-time)
- Hook waits for user decision
- Immediate feedback to Claude Code
- Requires timeout handling
- Best user experience

### Option 2: Asynchronous with Polling
- Hook returns immediately with "pending"
- Claude Code polls for decision
- More complex but scalable
- Good for multiple concurrent hooks

### Option 3: Hybrid Approach
- Critical tools (Bash, Edit) block synchronously
- Non-critical tools (Read, List) continue immediately
- Configurable per tool type
- Balanced approach

## Current Limitation

Our webhook endpoints are designed for fire-and-forget notifications, not real-time decision control. We need to refactor to support:

1. **Blocking requests** that wait for user input
2. **Timeout handling** for unresponsive users  
3. **Policy-based auto-approval** for trusted tools
4. **Real-time task status updates** from web interface

## Recommended Next Steps

1. **Implement synchronous PreToolUse handler** with decision waiting
2. **Add timeout configuration** for different hook types
3. **Create policy engine** for auto-approval rules
4. **Update web interface** for real-time decision making
5. **Add WebSocket support** for instant task updates

This would transform the system from a logging/notification system into a true remote control system for Claude Code.