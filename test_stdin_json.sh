#!/bin/bash

BASE_URL="http://localhost:10291"

echo "Testing Claude Code stdin JSON parsing..."
echo "========================================"

# Sample Claude Code JSON payloads based on the research
echo ""
echo "1. Testing PreToolUse with complete Claude Code JSON:"
echo '{
  "session_id": "abc123-def456-789",
  "transcript_path": "/Users/test/.claude/projects/test-project/transcript.jsonl",
  "cwd": "/Users/test/project",
  "hook_event_name": "PreToolUse",
  "tool_name": "Write", 
  "tool_input": {
    "file_path": "/path/to/file.txt",
    "content": "Hello World"
  }
}' | curl -X POST "$BASE_URL/webhook/pre-tool-use" \
    -H "Content-Type: application/json" \
    -d @- \
    -w "\nHTTP Status: %{http_code}\n" 2>/dev/null

echo ""
echo "2. Testing PostToolUse with Claude Code JSON:"
echo '{
  "session_id": "abc123-def456-789",
  "transcript_path": "/Users/test/.claude/projects/test-project/transcript.jsonl", 
  "cwd": "/Users/test/project",
  "hook_event_name": "PostToolUse",
  "tool_name": "Bash",
  "tool_input": {
    "command": "ls -la",
    "description": "List files"
  },
  "tool_output": "total 16\ndrwxr-xr-x  4 user  staff  128 Jul 24 14:00 .\ndrwxr-xr-x  3 user  staff   96 Jul 24 13:59 ..",
  "success": true
}' | curl -X POST "$BASE_URL/webhook/post-tool-use" \
    -H "Content-Type: application/json" \
    -d @- \
    -w "\nHTTP Status: %{http_code}\n" 2>/dev/null

echo ""
echo "3. Testing Notification with Claude Code JSON:"
echo '{
  "session_id": "abc123-def456-789",
  "transcript_path": "/Users/test/.claude/projects/test-project/transcript.jsonl",
  "cwd": "/Users/test/project", 
  "hook_event_name": "Notification",
  "message": "Claude needs your permission to continue"
}' | curl -X POST "$BASE_URL/webhook/notification" \
    -H "Content-Type: application/json" \
    -d @- \
    -w "\nHTTP Status: %{http_code}\n" 2>/dev/null

echo ""
echo "4. Testing UserPromptSubmit with Claude Code JSON:"
echo '{
  "session_id": "abc123-def456-789",
  "transcript_path": "/Users/test/.claude/projects/test-project/transcript.jsonl",
  "cwd": "/Users/test/project",
  "hook_event_name": "UserPromptSubmit", 
  "user_input": "Help me write a Python script"
}' | curl -X POST "$BASE_URL/webhook/user-prompt-submit" \
    -H "Content-Type: application/json" \
    -d @- \
    -w "\nHTTP Status: %{http_code}\n" 2>/dev/null

echo ""
echo "5. Testing PreCompact with Claude Code JSON:"
echo '{
  "session_id": "abc123-def456-789",
  "transcript_path": "/Users/test/.claude/projects/test-project/transcript.jsonl",
  "cwd": "/Users/test/project",
  "hook_event_name": "PreCompact",
  "compact_trigger": "manual"
}' | curl -X POST "$BASE_URL/webhook/pre-compact" \
    -H "Content-Type: application/json" \
    -d @- \
    -w "\nHTTP Status: %{http_code}\n" 2>/dev/null

echo ""
echo "Check the server logs to see parsed Claude Code data:"
echo "  make logs-server"
echo "  docker compose logs -f claude-control"