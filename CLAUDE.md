# Claude Memory - Haiper Project

## Tmux Session Management

### Naming Convention
Use format: `{project}-{purpose}` for consistent session organization:
- **project**: Short identifier (lowercase, hyphenated) 
- **purpose**: Session purpose (dev, debug, logs, test, etc.)

Examples:
- `haiper-dev` - Main development session
- `haiper-debug` - Debug session with server logs  
- `haiper-test` - Testing and validation
- `claude-code-dev` - Claude Code development

### Starting and Managing Sessions

#### Create Session:
```bash
# Create new session in project directory
tmux new-session -d -s haiper-dev -c /Users/dan/Software/haiper

# Create with specific window name
tmux new-session -d -s haiper-dev -n main -c /Users/dan/Software/haiper
```

#### Setup Development Windows:
```bash
# Create common windows
tmux new-window -t haiper-dev -n logs
tmux new-window -t haiper-dev -n docker  
tmux new-window -t haiper-dev -n test

# Send startup commands
tmux send-keys -t haiper-dev:logs "make logs-debug" Enter
tmux send-keys -t haiper-dev:docker "docker compose ps" Enter
tmux send-keys -t haiper-dev:main "cd /Users/dan/Software/haiper" Enter
```

#### Send Commands to Sessions:
```bash
# Send to main window
tmux send-keys -t haiper-dev "make debug-on" Enter

# Send to specific window  
tmux send-keys -t haiper-dev:logs "make logs-debug" Enter
tmux send-keys -t haiper-dev:docker "make status" Enter

# Send without pressing Enter (for editing)
tmux send-keys -t haiper-dev "make "
```

#### Session Management:
```bash
# List all sessions
tmux list-sessions

# Attach to session
tmux attach-session -t haiper-dev

# Kill session when done
tmux kill-session -t haiper-dev

# List windows in session
tmux list-windows -t haiper-dev
```

### Common Development Workflow

Typical haiper development setup:
1. **main** - Primary coding/editing window
2. **logs** - Debug server logs (`make logs-debug`)
3. **docker** - Container management (`make status`, `make up/down`)
4. **test** - Running tests and validation

### Key Commands for This Project
- `make debug-on` - Switch to debug mode
- `make debug-off` - Switch back to normal mode  
- `make logs-debug` - View debug server logs
- `make status` - Check container status
- `make up/down` - Start/stop services

### Tips
- Always check existing sessions with `tmux ls` before creating new ones
- Use descriptive window names within sessions
- Send commands with `tmux send-keys -t session:window "command" Enter`
- Clean up unused sessions regularly