# Test Debug Webhook Handler

This document explains how to use the test debug webhook handler to inspect the format of incoming Claude Code webhook requests.

## Purpose

The test debug handler allows you to see exactly what data Claude Code is sending in webhook requests without any processing. It provides nicely formatted logging output to help you understand:

- HTTP headers received
- JSON payload structure
- Key Claude Code fields
- Request metadata

## Available Debug Endpoints

The debug handler creates parallel endpoints to the regular webhook handlers:

### Debug Endpoints
- `POST /debug/webhook/pre-tool-use`
- `POST /debug/webhook/post-tool-use` 
- `POST /debug/webhook/notification`
- `POST /debug/webhook/user-prompt-submit`
- `POST /debug/webhook/stop`
- `POST /debug/webhook/subagent-stop`
- `POST /debug/webhook/pre-compact`
- `POST /debug/webhook/{hookType}` (generic handler)

### Regular Endpoints (for comparison)
- `POST /webhook/pre-tool-use`
- `POST /webhook/post-tool-use`
- `POST /webhook/notification`
- `POST /webhook/user-prompt-submit`
- `POST /webhook/stop`
- `POST /webhook/subagent-stop`
- `POST /webhook/pre-compact`

## How to Use

### Option 1: Docker Container (Recommended)

Start the lightweight debug server:
```bash
docker compose -f docker-compose.debug.yml up
```

The debug server will be available at:
- Debug endpoints: `http://localhost:9090/debug/webhook/`
- Info page: `http://localhost:9090/`
- Health check: `http://localhost:9090/health`

### Option 2: Local Go Server

Start the full server with debug routes:
```bash
go run cmd/server/main.go
```

The server will log that debug routes are registered:
```
✅ Test debug routes registered
🐛 Debug webhook endpoint: http://localhost:8080/debug/webhook/
```

### 1. Start the Server (Choose an option above)

### 2. Update hooks.json to Use Debug Endpoints

To test with the debug handler, temporarily modify your `hooks.json` file to point to the debug endpoints instead of the regular ones:

**Original hooks.json:**
```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "cat | curl -s -X POST http://localhost:10291/webhook/pre-tool-use -H 'Content-Type: application/json' -d @- --max-time 300"
          }
        ]
      }
    ]
  }
}
```

**Debug version:**
```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "cat | curl -s -X POST http://localhost:8080/debug/webhook/pre-tool-use -H 'Content-Type: application/json' -d @- --max-time 300"
          }
        ]
      }
    ]
  }
}
```

**Note:** Change the port from `10291` to `8080` (or whatever port your server runs on) and add `/debug` to the path.

### 3. Trigger Claude Code Actions

Once you've updated the hooks to use debug endpoints, perform actions in Claude Code that trigger webhooks. You'll see formatted debug output in the server logs.

## Example Debug Output

When a webhook is triggered, you'll see output like this:

```
╔══════════════════════════════════════════════════════════════════════════════════════════════
║ 🚀 CLAUDE CODE WEBHOOK DEBUG - 2024-01-15 14:30:25.123
║ Hook Type: PreToolUse  
╠══════════════════════════════════════════════════════════════════════════════════════════════
║ HTTP Method: POST
║ URL Path: /debug/webhook/pre-tool-use
║ Remote Address: 127.0.0.1:54321
║
║ 📋 HTTP HEADERS:
║   Content-Length: 1234
║   Content-Type: application/json
║   User-Agent: curl/7.68.0
║
║ 📦 REQUEST BODY:
║   JSON Data:
║   {
║     "hook_event_name": "PreToolUse",
║     "session_id": "123e4567-e89b-12d3-a456-426614174000",
║     "tool_name": "Bash",
║     "cwd": "/Users/dan/Software/haiper",
║     "command": "ls -la",
║     "description": "List files in directory"
║   }
║
║ 🔍 KEY CLAUDE CODE FIELDS:
║   hook_event_name: PreToolUse
║   session_id: 123e4567-e89b-12d3-a456-426614174000
║   tool_name: Bash
║   cwd: /Users/dan/Software/haiper
║   command: ls -la
║   description: List files in directory
║
║ 📊 REQUEST STATS:
║   Body Size: 1234 bytes
║   Content-Type: application/json
║   User-Agent: curl/7.68.0
╚══════════════════════════════════════════════════════════════════════════════════════════════
```

## Features

The debug handler provides:

1. **Nicely Formatted Output**: Easy-to-read boxed format with emojis
2. **Complete Request Details**: Headers, body, metadata
3. **JSON Pretty Printing**: Indented JSON for readability  
4. **Key Field Extraction**: Highlights important Claude Code fields
5. **Error Handling**: Shows malformed JSON gracefully
6. **Successful Response**: Returns proper Claude Code response format

## Response Format

The debug handler returns a successful JSON response that Claude Code expects:

```json
{
  "continue": true,
  "suppress_output": true,
  "debug": true,
  "message": "Request logged successfully"
}
```

This allows Claude Code to continue normal operation while you inspect the webhook data.

## Switching Back

After testing, remember to switch your `hooks.json` back to the regular webhook endpoints (without `/debug`) for normal operation.