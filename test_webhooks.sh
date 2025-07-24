#!/bin/bash

# Claude Hook Integration Tests
# Tests that webhook endpoints are working and properly creating tasks

set -e

BASE_URL="http://localhost:10291"
WEBHOOK_URL="$BASE_URL/webhook"

echo "🧪 Testing Claude Control Webhook Integration"
echo "============================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function to test webhook endpoint
test_webhook() {
    local hook_type="$1"
    local endpoint="$2"
    local test_data="$3"
    local description="$4"
    
    echo -e "\n${YELLOW}Testing: $description${NC}"
    echo "Endpoint: $WEBHOOK_URL/$endpoint"
    
    # Send webhook request
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$WEBHOOK_URL/$endpoint" \
        -H "Content-Type: application/json" \
        -d "$test_data")
    
    # Extract HTTP code
    http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d: -f2)
    response_body=$(echo "$response" | grep -v "HTTP_CODE:")
    
    echo "HTTP Code: $http_code"
    echo "Response: $response_body"
    
    if [ "$http_code" -eq 201 ]; then
        echo -e "${GREEN}✅ PASS${NC}: Webhook accepted and task created"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        
        # Extract task ID and verify it was created
        task_id=$(echo "$response_body" | grep -o '"task_id":"[^"]*"' | cut -d'"' -f4)
        if [ -n "$task_id" ]; then
            echo "Task ID: $task_id"
            
            # Verify task exists via API
            task_response=$(curl -s "$BASE_URL/api/tasks/$task_id")
            if echo "$task_response" | grep -q '"success":true'; then
                echo -e "${GREEN}✅ PASS${NC}: Task retrievable via API"
                TESTS_PASSED=$((TESTS_PASSED + 1))
            else
                echo -e "${RED}❌ FAIL${NC}: Task not retrievable via API"
                TESTS_FAILED=$((TESTS_FAILED + 1))
            fi
        else
            echo -e "${RED}❌ FAIL${NC}: No task_id in response"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        fi
    else
        echo -e "${RED}❌ FAIL${NC}: HTTP $http_code - Expected 201"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Test health endpoint first
echo -e "\n${YELLOW}Testing: Health Check${NC}"
health_response=$(curl -s -w "\nHTTP_CODE:%{http_code}" "$BASE_URL/health")
health_code=$(echo "$health_response" | grep "HTTP_CODE:" | cut -d: -f2)

if [ "$health_code" -eq 200 ]; then
    echo -e "${GREEN}✅ PASS${NC}: Server is healthy"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}❌ FAIL${NC}: Server health check failed (HTTP $health_code)"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    echo "Exiting - server not accessible"
    exit 1
fi

# Test PreToolUse webhook (most important for Claude Code)
test_webhook "PreToolUse" "pre-tool-use" '{
    "hook_type": "PreToolUse",
    "tool": "Bash",
    "command": "ls -la",
    "session_id": "test-session-123",
    "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)'"
}' "PreToolUse Hook (Tool Approval Required)"

# Test PostToolUse webhook
test_webhook "PostToolUse" "post-tool-use" '{
    "hook_type": "PostToolUse", 
    "tool": "Bash",
    "command": "ls -la",
    "output": "total 8\ndrwxr-xr-x  3 user  staff   96 Jul 23 14:30 .",
    "exit_code": 0,
    "session_id": "test-session-123"
}' "PostToolUse Hook (Tool Completed)"

# Test Notification webhook
test_webhook "Notification" "notification" '{
    "hook_type": "Notification",
    "message": "Claude needs permission to proceed",
    "level": "warning",
    "session_id": "test-session-123"
}' "Notification Hook (User Attention Required)"

# Test UserPromptSubmit webhook
test_webhook "UserPromptSubmit" "user-prompt-submit" '{
    "hook_type": "UserPromptSubmit",
    "prompt": "Help me write a Python script",
    "session_id": "test-session-123",
    "user_id": "test-user"
}' "UserPromptSubmit Hook (Prompt Validation)"

# Test Stop webhook
test_webhook "Stop" "stop" '{
    "hook_type": "Stop",
    "session_id": "test-session-123",
    "duration": 120.5,
    "reason": "completed"
}' "Stop Hook (Session Ended)"

# Test generic webhook endpoint
test_webhook "PreCompact" "PreCompact" '{
    "hook_type": "PreCompact",
    "matcher": "auto",
    "context_size": 15000,
    "session_id": "test-session-123"
}' "Generic Webhook Endpoint (PreCompact)"

# Test dashboard accessibility
echo -e "\n${YELLOW}Testing: Dashboard Access${NC}"
dashboard_response=$(curl -s -w "\nHTTP_CODE:%{http_code}" "$BASE_URL/dashboard")
dashboard_code=$(echo "$dashboard_response" | grep "HTTP_CODE:" | cut -d: -f2)

if [ "$dashboard_code" -eq 200 ]; then
    if echo "$dashboard_response" | grep -q "Claude Control Dashboard"; then
        echo -e "${GREEN}✅ PASS${NC}: Dashboard accessible and renders correctly"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}❌ FAIL${NC}: Dashboard accessible but content incorrect"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
else
    echo -e "${RED}❌ FAIL${NC}: Dashboard not accessible (HTTP $dashboard_code)"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test API tasks list
echo -e "\n${YELLOW}Testing: API Tasks List${NC}"
api_response=$(curl -s -w "\nHTTP_CODE:%{http_code}" "$BASE_URL/api/tasks")
api_code=$(echo "$api_response" | grep "HTTP_CODE:" | cut -d: -f2)

if [ "$api_code" -eq 200 ]; then
    if echo "$api_response" | grep -q '"success":true'; then
        echo -e "${GREEN}✅ PASS${NC}: API tasks endpoint working"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        
        # Count tasks created by our tests
        task_count=$(echo "$api_response" | grep -o '"tasks":\[' | wc -l)
        echo "Tasks in system: Checking response structure..."
    else
        echo -e "${RED}❌ FAIL${NC}: API tasks endpoint returned invalid response"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
else
    echo -e "${RED}❌ FAIL${NC}: API tasks endpoint not accessible (HTTP $api_code)"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Summary
echo -e "\n============================================"
echo "🧪 Test Results Summary"
echo "============================================"
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
echo "Total Tests: $((TESTS_PASSED + TESTS_FAILED))"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All tests passed! Claude Control System is working correctly.${NC}"
    echo ""
    echo "✅ Webhook endpoints are accepting Claude hook requests"
    echo "✅ Tasks are being created and stored in database"
    echo "✅ Web dashboard is accessible"
    echo "✅ API endpoints are responding correctly"
    echo ""
    exit 0
else
    echo -e "\n${RED}❌ Some tests failed. Check the output above for details.${NC}"
    echo ""
    echo "Common issues:"
    echo "- Ensure Docker services are running: docker-compose ps"
    echo "- Check service logs: docker-compose logs claude-control"
    echo "- Verify database connection: docker-compose logs postgres"
    exit 1
fi