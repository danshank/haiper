# Claude Code Hooks Setup Guide

This guide shows you how to configure Claude Code hooks to work with your Claude Control System.

## Prerequisites

- Claude Code installed and working
- Claude Control System running (see main README)
- Services accessible on your network

## Step 1: Identify Your Server URL

Based on your `.env` configuration, your webhook endpoint is:
```
http://localhost:10291/webhook/
```

For network access from other devices, replace `localhost` with your machine's IP address:
```
http://192.168.1.100:10291/webhook/
```

## Step 2: Configure Claude Code Hooks

Claude Code hooks can be configured in two ways:

### Option A: Blocking Hook Configuration (Recommended for Real-time Control)

Use the **blocking webhook endpoints** that wait for your decision before allowing Claude Code to continue:

Create or edit `~/.claude/settings.json`:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "RESPONSE=$(curl -s -X POST http://localhost:10291/webhook/pre-tool-use-blocking -H 'Content-Type: application/json' -d '{\"hook_type\": \"PreToolUse\", \"session_id\": \"'\"$session_id\"'\", \"tool\": \"'\"$tool_name\"'\", \"data\": '\"$tool_input\"'}' --max-time 300); echo \"$RESPONSE\"; if echo \"$RESPONSE\" | jq -e '.continue == false' > /dev/null; then exit 2; fi",
            "timeout": 300
          }
        ]
      }
    ],
    "Notification": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "RESPONSE=$(curl -s -X POST http://localhost:10291/webhook/notification-blocking -H 'Content-Type: application/json' -d '{\"hook_type\": \"Notification\", \"session_id\": \"'\"$session_id\"'\", \"message\": \"Claude needs your attention\"}' --max-time 300); echo \"$RESPONSE\"; if echo \"$RESPONSE\" | jq -e '.continue == false' > /dev/null; then exit 2; fi",
            "timeout": 300
          }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "RESPONSE=$(curl -s -X POST http://localhost:10291/webhook/user-prompt-submit-blocking -H 'Content-Type: application/json' -d '{\"hook_type\": \"UserPromptSubmit\", \"session_id\": \"'\"$session_id\"'\", \"prompt\": '\"$user_input\"'}' --max-time 300); echo \"$RESPONSE\"; if echo \"$RESPONSE\" | jq -e '.continue == false' > /dev/null; then exit 2; fi",
            "timeout": 300
          }
        ]
      }
    ]
  }
}
```

### Option B: Non-blocking Hook Configuration (For Logging Only)

Use the **non-blocking webhook endpoints** that just log events without waiting:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "curl -s -X POST http://localhost:10291/webhook/pre-tool-use -H 'Content-Type: application/json' -d '{\"hook_type\": \"PreToolUse\", \"session_id\": \"'\"$session_id\"'\", \"tool\": \"'\"$tool_name\"'\", \"data\": '\"$tool_input\"'}' --max-time 10 || true"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "curl -s -X POST http://localhost:10291/webhook/post-tool-use -H 'Content-Type: application/json' -d '{\"hook_type\": \"PostToolUse\", \"session_id\": \"'\"$session_id\"'\", \"tool\": \"'\"$tool_name\"'\", \"success\": '\"$success\"', \"data\": '\"$tool_output\"'}' --max-time 10 || true"
          }
        ]
      }
    ],
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "curl -s -X POST http://localhost:10291/webhook/stop -H 'Content-Type: application/json' -d '{\"hook_type\": \"Stop\", \"session_id\": \"'\"$session_id\"'\"}' --max-time 10 || true"
          }
        ]
      }
    ]
  }
}
```

### Option B: Using settings.json

Create or edit `~/.claude/settings.json`:

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "curl -X POST http://localhost:10291/webhook/pre-tool-use -H \"Content-Type: application/json\" -d \"{\\\"hook_type\\\": \\\"PreToolUse\\\", \\\"tool\\\": \\\"$CLAUDE_TOOL_NAME\\\", \\\"data\\\": $CLAUDE_HOOK_DATA}\" --silent --max-time 10"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "curl -X POST http://localhost:10291/webhook/post-tool-use -H \"Content-Type: application/json\" -d \"{\\\"hook_type\\\": \\\"PostToolUse\\\", \\\"tool\\\": \\\"$CLAUDE_TOOL_NAME\\\", \\\"data\\\": $CLAUDE_HOOK_DATA}\" --silent --max-time 10"
          }
        ]
      }
    ],
    "Notification": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "curl -X POST http://localhost:10291/webhook/notification -H \"Content-Type: application/json\" -d \"{\\\"hook_type\\\": \\\"Notification\\\", \\\"data\\\": $CLAUDE_HOOK_DATA}\" --silent --max-time 10"
          }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "curl -X POST http://localhost:10291/webhook/user-prompt-submit -H \"Content-Type: application/json\" -d \"{\\\"hook_type\\\": \\\"UserPromptSubmit\\\", \\\"data\\\": $CLAUDE_HOOK_DATA}\" --silent --max-time 10"
          }
        ]
      }
    ],
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "curl -X POST http://localhost:10291/webhook/stop -H \"Content-Type: application/json\" -d \"{\\\"hook_type\\\": \\\"Stop\\\", \\\"data\\\": $CLAUDE_HOOK_DATA}\" --silent --max-time 10"
          }
        ]
      }
    ],
    "SubagentStop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",  
            "command": "curl -X POST http://localhost:10291/webhook/subagent-stop -H \"Content-Type: application/json\" -d \"{\\\"hook_type\\\": \\\"SubagentStop\\\", \\\"data\\\": $CLAUDE_HOOK_DATA}\" --silent --max-time 10"
          }
        ]
      }
    ],
    "PreCompact": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "curl -X POST http://localhost:10291/webhook/pre-compact -H \"Content-Type: application/json\" -d \"{\\\"hook_type\\\": \\\"PreCompact\\\", \\\"matcher\\\": \\\"$CLAUDE_COMPACT_TRIGGER\\\", \\\"data\\\": $CLAUDE_HOOK_DATA}\" --silent --max-time 10"
          }
        ]
      }
    ]
  }
}
```

## Step 3: Network Configuration (For Remote Access)

If you want to access the system from your phone over your local network or VPN:

1. **Find your machine's IP address:**
   ```bash
   # On macOS/Linux
   ifconfig | grep "inet " | grep -v 127.0.0.1
   
   # Or
   ip addr show | grep "inet " | grep -v 127.0.0.1
   ```

2. **Update the webhook URLs in your hooks configuration:**
   Replace `localhost` with your actual IP address:
   ```
   http://192.168.1.100:10291/webhook/pre-tool-use
   ```

3. **Update your `.env` file:**
   ```bash
   # Set this to your actual network address
   WEB_DOMAIN=192.168.1.100:10291
   ```

4. **Restart the services:**
   ```bash
   docker-compose down && docker-compose up -d
   ```

## Step 4: TMux Session Setup

For the system to send commands back to Claude Code, you need a tmux session:

```bash
# Create the session that matches your configuration
tmux new-session -d -s claude-code-session

# Start Claude Code in that session
tmux send-keys -t claude-code-session "claude" Enter
```

## Step 5: Testing the Setup

1. **Test webhook connectivity:**
   ```bash
   curl -X POST http://localhost:10291/webhook/notification \
     -H "Content-Type: application/json" \
     -d '{"hook_type": "Notification", "data": {"test": "hello"}}'
   ```

2. **Check the dashboard:**
   Open `http://localhost:10291/dashboard` in your browser

3. **Test with Claude Code:**
   Run any command in Claude Code and check if tasks appear in the dashboard

## Step 6: Mobile Access Setup

1. **Install NTFY app on your phone:**
   - iOS: Search "ntfy" in App Store
   - Android: Search "ntfy" in Google Play Store

2. **Configure NTFY subscription:**
   - Server URL: `http://your-ip:10292`
   - Topic: `claude-notifications`

3. **Test notification:**
   ```bash
   curl -X POST http://localhost:10292 \
     -H "Content-Type: application/json" \
     -d '{
       "topic": "claude-notifications",
       "title": "Test Notification",
       "message": "Claude Control System is working!",
       "click": "http://your-ip:10291/dashboard"
     }'
   ```

## Customization Options

### Selective Hook Configuration

You can configure hooks for specific tools only:

```toml
[[hooks]]
event = "PreToolUse"
matcher = "Bash"  # Only for Bash tool
command = "curl ..."

[[hooks]]
event = "PreToolUse"
matcher = "Edit"  # Only for Edit tool
command = "curl ..."
```

### Hook with Custom Data

Add custom metadata to your hooks:

```toml
[[hooks]]
event = "PreToolUse"
command = '''curl -X POST http://localhost:10291/webhook/pre-tool-use \
  -H "Content-Type: application/json" \
  -d "{\"hook_type\": \"PreToolUse\", \"tool\": \"$CLAUDE_TOOL_NAME\", \"custom_field\": \"my-value\", \"data\": $CLAUDE_HOOK_DATA}"'''
```

## Troubleshooting

### Hooks Not Firing
- Check Claude Code configuration with `/hooks` command
- Verify webhook URL is accessible: `curl http://localhost:10291/health`
- Check Claude Control logs: `docker-compose logs claude-control`

### Notifications Not Working
- Test NTFY directly: `curl http://localhost:10292/v1/health`
- Check notification topic matches configuration
- Verify phone app is subscribed to correct server and topic

### TMux Commands Not Working
- Ensure tmux session exists: `tmux list-sessions`
- Check session name matches configuration
- Verify Claude Code is running in the correct tmux session

### Network Access Issues
- Check firewall settings
- Verify IP address is correct
- Test connectivity: `telnet your-ip 10291`

## Security Considerations

- The system is designed for local network use
- Consider using HTTPS in production environments
- Restrict access to webhook endpoints if needed
- Use strong passwords for database access
- Keep Docker images updated

## Advanced Configuration

### Environment Variables
All webhook URLs and configuration can be customized via environment variables in your `.env` file.

### Custom Actions
The system supports custom action types beyond the standard approve/reject/continue actions.

### Database Queries
You can query the PostgreSQL database directly for advanced task analysis:
```sql
-- Connect to database
psql -h localhost -p 10293 -U claude_user -d claude_control

-- View all tasks
SELECT id, hook_type, status, created_at FROM tasks ORDER BY created_at DESC;
```

## Support

If you encounter issues:
1. Check the logs: `docker-compose logs`
2. Verify all services are healthy: `docker-compose ps`
3. Test webhook connectivity manually
4. Check Claude Code hooks configuration with `/hooks`