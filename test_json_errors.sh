#!/bin/bash

BASE_URL="http://localhost:10291"

echo "Testing JSON error handling with detailed logging..."
echo "================================================="

# Start the server in background (if not already running)
if ! curl -s "$BASE_URL/health" > /dev/null 2>&1; then
    echo "Starting server for testing..."
    make up-server > /dev/null 2>&1 &
    sleep 3
fi

echo ""
echo "1. Testing completely malformed JSON:"
curl -X POST "$BASE_URL/webhook/pre-tool-use" \
    -H "Content-Type: application/json" \
    -d '{"hook_type": "PreToolUse", "malformed": json}' \
    -w "\nHTTP Status: %{http_code}\n" 2>/dev/null
echo ""

echo "2. Testing wrong data type:"
curl -X POST "$BASE_URL/webhook/pre-tool-use" \
    -H "Content-Type: application/json" \
    -d '{"hook_type": 12345, "tool": "Bash"}' \
    -w "\nHTTP Status: %{http_code}\n" 2>/dev/null
echo ""

echo "3. Testing invalid hook type:"
curl -X POST "$BASE_URL/webhook/pre-tool-use" \
    -H "Content-Type: application/json" \
    -d '{"hook_type": "InvalidHookType", "tool": "Bash"}' \
    -w "\nHTTP Status: %{http_code}\n" 2>/dev/null
echo ""

echo "4. Testing empty body:"
curl -X POST "$BASE_URL/webhook/pre-tool-use" \
    -H "Content-Type: application/json" \
    -d '' \
    -w "\nHTTP Status: %{http_code}\n" 2>/dev/null
echo ""

echo "5. Testing valid JSON (should succeed):"
curl -X POST "$BASE_URL/webhook/pre-tool-use" \
    -H "Content-Type: application/json" \
    -d '{"hook_type": "PreToolUse", "tool": "Bash", "session_id": "test-123"}' \
    -w "\nHTTP Status: %{http_code}\n" 2>/dev/null
echo ""

echo "Check the server logs with: make logs-server"
echo "Or: docker compose logs -f claude-control"