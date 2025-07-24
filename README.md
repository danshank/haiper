# Claude Code Remote Control System

This is a description of the setup that I want to achieve for my general coding workflow.

The goals are to set up a local server that is able to listen via hooks to output that Claude code is putting off as I ask it to run different coding tasks.

Once it detects the output of one of those hooks, it's going to send a notification via NTFY (or notify) to my phone with a link that includes information about the notification, I will be able to go to that web page, which I'm hosting on my local network and remote it in via VPN.

I'll be able to visit that webpage and tell claude code via just a simple HTTP request. What I wanted to do next.

So what is it going to involve is a VPN as the network layer.

And I already have that set up.

I can already via my phone, access the same VPN that my workstation is hooked onto.

I also need to have a go server running. So that I can actually grab the hooks that Claude is going to be triggering.

And basically sending a HDP request to that web server that says, hey , this is ready. As well as sending via notify that notification to my phone, the general structure will be that if a clod code has to get feedback from me, it will send both a notification and along with that notification, there should be some ID attached to it .

That ID is then I can then use that ID to create a webpage and go.

It will contain the text, that notification as well as follow-up actions that I can take if I decide to give feedback or take a specific action as follow-up, then the go server will return a JSON response directly to Claude Code indicating whether to continue, stop, or take other actions.

So it's essentially an integrated webpage on my local network that I can use to go back and forth with Claude Code and control it via my phone using Claude Code's native JSON hook response system.

## System Architecture

```mermaid
graph TB
    %% User Devices
    subgraph "Mobile Device"
        Phone["📱 Your Phone"]
        Browser["🌐 Web Browser<br/>• View Tasks<br/>• Approve/Reject<br/>• Continue Actions"]
        Phone --> Browser
    end
    
    subgraph "Workstation"
        Claude["🤖 Claude Code<br/>(JSON Hook Responses)"]
        GoServer["⚙️ Go Server<br/>• Webhook Handlers<br/>• Web Interface<br/>• JSON Response Builder<br/>• NTFY Integration"]
        DB[("🗄️ PostgreSQL<br/>• Task Data<br/>• History")]
        NTFY["📢 NTFY Server<br/>• Push Notifications"]
    end
    
    %% Network Layer
    subgraph "Network"
        Tailscale["🔒 Tailscale VPN"]
    end
    
    %% Connections
    Phone -.->|VPN Access| Tailscale
    Tailscale -.->|Secure Tunnel| GoServer
    
    %% Claude Code Flow
    Claude -->|Webhook Request| GoServer
    GoServer -->|Store Task| DB
    GoServer -->|Send Notification| NTFY
    GoServer -->|JSON Response| Claude
    NTFY -->|Push Notification| Phone
    
    %% User Interaction Flow
    Browser -->|HTTP Request| GoServer
    GoServer -->|Retrieve Task| DB
    GoServer -->|Update Task| DB
    
    %% Docker Services
    subgraph "Docker Containers"
        DB
        NTFY
    end
    
    %% Styling
    classDef userDevice fill:#e1f5fe,stroke:#01579b,stroke-width:2px
    classDef workstation fill:#f3e5f5,stroke:#4a148c,stroke-width:2px
    classDef network fill:#e8f5e8,stroke:#1b5e20,stroke-width:2px
    classDef docker fill:#fff3e0,stroke:#e65100,stroke-width:2px
    
    class Phone,Browser userDevice
    class Claude,GoServer workstation
    class Tailscale network
    class DB,NTFY docker
```

### Data Flow Sequence

```mermaid
sequenceDiagram
    participant C as Claude Code
    participant G as Go Server
    participant D as PostgreSQL
    participant N as NTFY
    participant P as Your Phone
    participant B as Web Browser
    
    Note over C,P: 1. Claude Code Hook Trigger
    C->>G: POST /webhook/{hook-type}
    G->>D: Store task data
    G->>N: Send notification with task link
    N->>P: Push notification
    
    Note over P,B: 2. User Interaction
    P->>B: Open notification link
    B->>G: GET /task/{task-id}
    G->>D: Retrieve task data
    G->>B: Return task details & actions
    
    Note over B,C: 3. Action Response
    B->>G: POST /task/{task-id}/action
    G->>D: Update task status
    Note over G,C: Future webhook requests receive user decision
    C->>G: Next webhook (if blocking)
    G->>C: JSON response with user decision
```

## Data Flow

### 1. Claude Code Hook Trigger

```
Claude Code → Webhook → Go Server
```

### 2. Notification & Storage

```
Go Server → PostgreSQL (store task)
Go Server → NTFY → Your Phone
```

### 3. User Interaction

```
Your Phone → Web Browser → Go Server → PostgreSQL (retrieve task)
```

### 4. Action Response

```
Your Phone → Web Browser → Go Server → JSON Response → Claude Code
```

## Implementation Plan

### 1. Go Server Components

#### Webhook Handlers
- **PreToolUse**: Before Claude executes a tool
- **PostToolUse**: After Claude completes a tool execution  
- **Notification**: When Claude needs user attention
- **Stop**: When Claude finishes a response

#### Web Interface
- Task dashboard showing all pending tasks
- Individual task view with action buttons
- Real-time updates via WebSocket (optional)

#### JSON Hook Response System
- Return Claude Code compliant JSON responses with `continue`, `stopReason`, and `suppressOutput` fields
- Support for blocking and non-blocking webhook responses
- Real-time decision management via Go channels

### 2. Database Schema

```sql
-- Tasks table
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hook_type VARCHAR(50) NOT NULL, -- PreToolUse, PostToolUse, Notification, Stop
    task_data JSONB NOT NULL, -- All hook data
    status VARCHAR(20) DEFAULT 'pending', -- pending, approved, rejected, completed
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    action_taken VARCHAR(50), -- approve, reject, continue, etc.
    response_data JSONB -- User's response/feedback
);

-- Task history for audit trail
CREATE TABLE task_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID REFERENCES tasks(id),
    action VARCHAR(50) NOT NULL,
    data JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### 3. Environment Configuration

```bash
# .env file
SERVER_PORT=8080
DATABASE_URL=postgresql://user:password@localhost:5432/claude_control
NTFY_SERVER_URL=http://localhost:80
NTFY_TOPIC=claude-notifications
WEB_DOMAIN=your-tailscale-ip:8080
```

### 4. Claude Code Hook Configuration

Configure Claude Code hooks using the `/hooks` command or by editing `~/.claude/settings.json`:

#### Option A: Interactive Configuration (Recommended)
```bash
# In Claude Code
/hooks

# Follow the interactive menu to add hooks for:
# - PreToolUse: Before tool execution
# - PostToolUse: After tool execution  
# - Notification: When Claude needs attention
# - UserPromptSubmit: When user submits input
# - Stop: When Claude finishes responding
# - SubagentStop: When subagent completes
# - PreCompact: Before context compaction
```

#### Option B: Manual JSON Configuration
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
            "command": "curl -s -X POST http://localhost:10291/webhook/pre-tool-use -H 'Content-Type: application/json' -d '{\"hook_type\": \"PreToolUse\", \"session_id\": \"'\"$session_id\"'\", \"tool\": \"'\"$tool_name\"'\", \"cwd\": \"'\"$cwd\"'\", \"data\": '\"$tool_input\"'}' --max-time 300"
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
            "command": "curl -s -X POST http://localhost:10291/webhook/post-tool-use -H 'Content-Type: application/json' -d '{\"hook_type\": \"PostToolUse\", \"session_id\": \"'\"$session_id\"'\", \"tool\": \"'\"$tool_name\"'\", \"cwd\": \"'\"$cwd\"'\", \"success\": '\"$success\"', \"data\": '\"$tool_output\"'}' --max-time 30"
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
            "command": "curl -s -X POST http://localhost:10291/webhook/notification -H 'Content-Type: application/json' -d '{\"hook_type\": \"Notification\", \"session_id\": \"'\"$session_id\"'\", \"cwd\": \"'\"$cwd\"'\", \"message\": \"Claude needs your attention\"}' --max-time 300"
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
            "command": "curl -s -X POST http://localhost:10291/webhook/user-prompt-submit -H 'Content-Type: application/json' -d '{\"hook_type\": \"UserPromptSubmit\", \"session_id\": \"'\"$session_id\"'\", \"cwd\": \"'\"$cwd\"'\", \"prompt\": '\"$user_input\"'}' --max-time 300"
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
            "command": "curl -s -X POST http://localhost:10291/webhook/stop -H 'Content-Type: application/json' -d '{\"hook_type\": \"Stop\", \"session_id\": \"'\"$session_id\"'\", \"cwd\": \"'\"$cwd\"'\"}' --max-time 30"
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
            "command": "curl -s -X POST http://localhost:10291/webhook/subagent-stop -H 'Content-Type: application/json' -d '{\"hook_type\": \"SubagentStop\", \"session_id\": \"'\"$session_id\"'\", \"cwd\": \"'\"$cwd\"'\"}' --max-time 30"
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
            "command": "curl -s -X POST http://localhost:10291/webhook/pre-compact -H 'Content-Type: application/json' -d '{\"hook_type\": \"PreCompact\", \"session_id\": \"'\"$session_id\"'\", \"cwd\": \"'\"$cwd\"'\", \"trigger\": \"'\"$compact_trigger\"'\"}' --max-time 30"
          }
        ]
      }
    ]
  }
}
```

#### Available Hook Variables
Claude Code provides these environment variables to hooks:
- `$session_id`: Unique session identifier
- `$transcript_path`: Path to session transcript  
- `$cwd`: Current working directory
- `$tool_name`: Name of the tool being used (PreToolUse/PostToolUse)
- `$tool_input`: JSON input to the tool (PreToolUse)
- `$tool_output`: JSON output from the tool (PostToolUse)
- `$success`: Tool execution success status (PostToolUse)
- `$user_input`: User's submitted prompt (UserPromptSubmit)
- `$compact_trigger`: What triggered compaction - "manual" or "auto" (PreCompact)

#### Network Configuration
Update the webhook URLs to match your actual server address:
- **Local**: `http://localhost:10291/webhook/`
- **Network**: `http://YOUR_IP:10291/webhook/`
- **VPN**: `http://YOUR_VPN_IP:10291/webhook/`

#### Selective Hook Configuration
To only hook specific tools, use matchers:
```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",  // Only Bash commands
        "hooks": [...]
      },
      {
        "matcher": "Edit|Write",  // Only Edit or Write tools
        "hooks": [...]
      }
    ]
  }
}
```

### 5. Docker Compose Setup

#### Quick Start

```bash
# Start all services
docker compose up -d

# Check status
docker compose ps

# View logs
docker compose logs -f

# Stop all services
docker compose down
```

#### Individual Container Management

```bash
# Start specific services
docker compose up -d postgres ntfy  # Start only database and notifications
docker compose up -d claude-control # Start only the main server

# Stop specific services
docker compose stop claude-control  # Stop only the main server
docker compose stop postgres        # Stop only the database

# Restart specific services
docker compose restart claude-control

# View logs for specific service
docker compose logs -f claude-control
docker compose logs -f postgres
docker compose logs -f ntfy

# Execute commands in running containers
docker compose exec claude-control /bin/sh
docker compose exec postgres psql -U claude_user -d claude_control

# Check service health
docker compose ps
```

#### Development Workflow

```bash
# Rebuild and restart after code changes
docker compose down
docker compose build claude-control
docker compose up -d

# Or rebuild and restart in one command
docker compose up -d --build claude-control

# View real-time logs during development
docker compose logs -f claude-control
```

#### Troubleshooting Commands

```bash
# Check container resource usage
docker compose top

# Inspect service configuration
docker compose config

# Remove all containers and volumes (destructive)
docker compose down -v

# View detailed container information
docker compose ps -a
docker inspect $(docker compose ps -q claude-control)
```

#### Environment Variables

Create a `.env` file to customize ports and configuration:

```bash
# .env
SERVER_PORT=8080
WEB_HOST_PORT=8080
POSTGRES_HOST_PORT=5432
NTFY_HOST_PORT=80
NTFY_TOPIC=claude-notifications
WEB_DOMAIN=localhost:8080
```

## JSON Hook Response System

This system uses Claude Code's native JSON hook response format instead of external TMux control. The server returns JSON responses that Claude Code interprets directly:

### Response Format

```json
{
  "continue": true,           // Allow Claude Code to continue (true/false)
  "stopReason": "string",     // Optional reason for stopping (if continue: false)
  "suppressOutput": false     // Hide hook output in Claude Code (true/false)
}
```

### Response Types

- **Approved Response**: `{"continue": true}`
- **Rejected Response**: `{"continue": false, "stopReason": "User rejected this action"}`
- **Timeout Response**: `{"continue": false, "stopReason": "Timeout: No user response within 5 minutes"}`
- **Suppressed Response**: `{"continue": true, "suppressOutput": true}`

### Blocking vs Non-Blocking Hooks

- **Blocking Hooks** (PreToolUse, Notification, UserPromptSubmit): Wait for user decision, return appropriate JSON response
- **Non-Blocking Hooks** (PostToolUse, Stop, etc.): Immediately return `{"continue": true, "suppressOutput": true}`

## Next Steps

1. **Testing**: Test the full flow from Claude → Webhook → JSON Response → Claude Code
2. **User Interface**: Enhance the web interface for better mobile experience
3. **Claude Hook Configuration**: Set up the hook configuration in Claude Code
4. **Deployment**: Deploy using Docker Compose and configure Tailscale access
5. **Documentation**: Complete API documentation and troubleshooting guides

## Key Features

- **Real-time Notifications**: Instant push notifications to your phone
- **Web Interface**: Clean, responsive web UI for task management
- **TMux Integration**: Direct control of Claude Code session
- **Task Persistence**: Complete history and audit trail
- **Flexible Actions**: Support for approve, reject, continue, and custom actions
- **VPN Access**: Secure access via Tailscale from anywhere 