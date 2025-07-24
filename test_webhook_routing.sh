#!/bin/bash

BASE_URL="http://localhost:10291"

echo "Testing New Webhook Routing Configuration"
echo "========================================"
echo "Blocking:     Notification, Stop, SubagentStop (create tasks, wait for user)"
echo "Non-blocking: PreToolUse, PostToolUse, UserPromptSubmit, PreCompact (immediate response)"
echo ""

# Start the server if not running
if ! curl -s "$BASE_URL/health" > /dev/null 2>&1; then
    echo "⚠️  Server not running. Start with: make up-server"
    exit 1
fi

echo "🧪 Testing Non-Blocking Webhooks (should return immediately):"
echo "============================================================"

echo ""
echo "1. PreToolUse (non-blocking):"
time curl -X POST "$BASE_URL/webhook/pre-tool-use" \
    -H "Content-Type: application/json" \
    -d '{"hook_event_name": "PreToolUse", "tool_name": "Bash", "session_id": "test-123"}' \
    -w "\nHTTP Status: %{http_code} | Time: %{time_total}s\n" 2>/dev/null

echo ""
echo "2. PostToolUse (non-blocking):"
time curl -X POST "$BASE_URL/webhook/post-tool-use" \
    -H "Content-Type: application/json" \
    -d '{"hook_event_name": "PostToolUse", "tool_name": "Edit", "session_id": "test-123"}' \
    -w "\nHTTP Status: %{http_code} | Time: %{time_total}s\n" 2>/dev/null

echo ""
echo "3. UserPromptSubmit (non-blocking):"
time curl -X POST "$BASE_URL/webhook/user-prompt-submit" \
    -H "Content-Type: application/json" \
    -d '{"hook_event_name": "UserPromptSubmit", "user_input": "Hello", "session_id": "test-123"}' \
    -w "\nHTTP Status: %{http_code} | Time: %{time_total}s\n" 2>/dev/null

echo ""
echo "4. PreCompact (non-blocking):"
time curl -X POST "$BASE_URL/webhook/pre-compact" \
    -H "Content-Type: application/json" \
    -d '{"hook_event_name": "PreCompact", "compact_trigger": "manual", "session_id": "test-123"}' \
    -w "\nHTTP Status: %{http_code} | Time: %{time_total}s\n" 2>/dev/null

echo ""
echo "🔒 Testing Blocking Webhooks (these will create tasks and may timeout):"
echo "======================================================================"
echo "⚠️  Note: These will wait for user decisions - expect longer response times"

echo ""
echo "5. Notification (blocking - will wait for user decision):"
echo "   This should create a task and send a notification to your phone"
curl -X POST "$BASE_URL/webhook/notification" \
    -H "Content-Type: application/json" \
    -d '{"hook_event_name": "Notification", "message": "Test notification", "session_id": "test-123"}' \
    -w "\nHTTP Status: %{http_code}\n" \
    --max-time 10 2>/dev/null || echo "⏱️  Request timed out (expected for blocking webhook)"

echo ""
echo "6. Stop (blocking - will wait for user decision):"
curl -X POST "$BASE_URL/webhook/stop" \
    -H "Content-Type: application/json" \
    -d '{"hook_event_name": "Stop", "session_id": "test-123"}' \
    -w "\nHTTP Status: %{http_code}\n" \
    --max-time 10 2>/dev/null || echo "⏱️  Request timed out (expected for blocking webhook)"

echo ""
echo "7. SubagentStop (blocking - will wait for user decision):"
curl -X POST "$BASE_URL/webhook/subagent-stop" \
    -H "Content-Type: application/json" \
    -d '{"hook_event_name": "SubagentStop", "session_id": "test-123"}' \
    -w "\nHTTP Status: %{http_code}\n" \
    --max-time 10 2>/dev/null || echo "⏱️  Request timed out (expected for blocking webhook)"

echo ""
echo "📊 Check database for created tasks:"
echo "  make db-status"
echo "  make db-recent-tasks"
echo ""
echo "📱 Check for notifications on your phone (if NTFY configured)"
echo ""
echo "🎯 Expected Results:"
echo "  - Non-blocking webhooks: Immediate {\"continue\": true, \"suppressOutput\": true}"
echo "  - Blocking webhooks: Create tasks, send notifications, wait for user decisions"