# Claude Control Server Makefile
# Provides convenient commands for Docker Compose management

.PHONY: help up down restart logs build clean status shell-server shell-db test db-clean-pending db-clean-all-tasks db-status db-recent-tasks db-shell health debug-on debug-off logs-debug

# Default target
help:
	@echo "Claude Control Server - Docker Compose Management"
	@echo ""
	@echo "Available commands:"
	@echo "  make up          - Start all services"
	@echo "  make down        - Stop all services"
	@echo "  make restart     - Restart all services"
	@echo "  make logs        - View logs from all services"
	@echo "  make build       - Build/rebuild the main server"
	@echo "  make clean       - Stop and remove all containers and volumes"
	@echo "  make status      - Show status of all containers"
	@echo ""
	@echo "Individual service commands:"
	@echo "  make up-db       - Start only database"
	@echo "  make up-ntfy     - Start only notifications"
	@echo "  make up-server   - Start only main server"
	@echo "  make restart-server - Restart only main server"
	@echo ""
	@echo "Development commands:"
	@echo "  make dev         - Build and start for development"
	@echo "  make logs-server - View only server logs"
	@echo "  make logs-db     - View only database logs"
	@echo "  make logs-ntfy   - View only NTFY logs"
	@echo ""
	@echo "Database commands:"
	@echo "  make db-clean-pending - Remove all pending tasks from database"
	@echo "  make db-clean-all-tasks - Remove ALL tasks and history (with confirmation)"
	@echo "  make db-status   - Show task counts by status"
	@echo "  make db-recent-tasks - Show last 10 tasks"
	@echo "  make db-shell    - Open PostgreSQL shell"
	@echo ""
	@echo "Utility commands:"
	@echo "  make shell-server - Open shell in server container"
	@echo "  make test        - Run a test webhook request"
	@echo "  make health      - Check service health status"
	@echo ""
	@echo "Debug commands:"
	@echo "  make debug-on    - Switch to debug server (saves current Claude config)"
	@echo "  make debug-off   - Switch back to normal server (restores Claude config)"
	@echo "  make logs-debug  - View debug server logs"

# Helper functions to get dynamic ports
get-server-port:
	@docker compose port claude-control 8080 2>/dev/null | cut -d: -f2 || echo "8080"

get-ntfy-port:
	@docker compose port ntfy 80 2>/dev/null | cut -d: -f2 || echo "80"

get-debug-port:
	@docker compose -f docker-compose.debug.yml port debug-server 8080 2>/dev/null | cut -d: -f2 || echo "10291"

# Main commands
up:
	docker compose up -d
	@echo "✅ All services started"
	@echo "📱 Dashboard: http://localhost:$$(make -s get-server-port)/dashboard"
	@echo "🔗 Webhook: http://localhost:$$(make -s get-server-port)/webhook/"

down:
	docker compose down
	@echo "🛑 All services stopped"

restart: down up

logs:
	docker compose logs -f

build:
	docker compose build claude-control
	@echo "🔨 Server image rebuilt"

clean:
	docker compose down -v
	docker system prune -f
	@echo "🧹 All containers, volumes, and unused images removed"

status:
	docker compose ps

# Individual service commands
up-db:
	docker compose up -d postgres
	@echo "✅ Database started"

up-ntfy:
	docker compose up -d ntfy
	@echo "✅ NTFY notification service started"

up-server:
	docker compose up -d claude-control
	@echo "✅ Claude Control server started"

restart-server:
	docker compose restart claude-control
	@echo "🔄 Server restarted"

# Development commands
dev: build up
	@echo "🚀 Development environment ready"
	@make logs-server

logs-server:
	docker compose logs -f claude-control

logs-db:
	docker compose logs -f postgres

logs-ntfy:
	docker compose logs -f ntfy

logs-debug:
	docker compose -f docker-compose.debug.yml logs -f debug-server

# Utility commands
shell-server:
	docker compose exec claude-control /bin/sh

shell-db:
	docker compose exec postgres psql -U claude_user -d claude_control

test:
	@echo "🧪 Testing webhook endpoint..."
	curl -X POST http://localhost:$$(make -s get-server-port)/webhook/pre-tool-use \
		-H "Content-Type: application/json" \
		-d '{"hook_type": "PreToolUse", "tool": "test", "test": true}' \
		|| echo "❌ Test failed - is the server running?"

# Database management commands
db-clean-pending:
	@echo "🧹 Cleaning pending tasks from database..."
	@docker compose exec postgres psql -U claude_user -d claude_control -c "DELETE FROM tasks WHERE status = 'pending';" || echo "❌ Failed to clean pending tasks"
	@echo "✅ Pending tasks cleared"

db-clean-all-tasks:
	@echo "🧹 Cleaning ALL tasks from database..."
	@read -p "Are you sure? This will delete all tasks and history (y/N): " confirm && [ "$$confirm" = "y" ] || exit 1
	@docker compose exec postgres psql -U claude_user -d claude_control -c "DELETE FROM task_history; DELETE FROM tasks;" || echo "❌ Failed to clean all tasks"
	@echo "✅ All tasks and history cleared"

db-status:
	@echo "📊 Database task status:"
	@docker compose exec postgres psql -U claude_user -d claude_control -c "SELECT status, COUNT(*) as count FROM tasks GROUP BY status ORDER BY status;"

db-recent-tasks:
	@echo "📋 Recent tasks (last 10):"
	@docker compose exec postgres psql -U claude_user -d claude_control -c "SELECT id, hook_type, status, created_at FROM tasks ORDER BY created_at DESC LIMIT 10;"

db-shell:
	@echo "🐘 Opening PostgreSQL shell..."
	@docker compose exec postgres psql -U claude_user -d claude_control

# Health checks
health:
	@echo "🔍 Checking service health..."
	@docker compose ps
	@echo ""
	@echo "Testing endpoints..."
	@curl -s http://localhost:$$(make -s get-server-port)/health > /dev/null && echo "✅ Server health OK" || echo "❌ Server health failed"
	@curl -s http://localhost:$$(make -s get-ntfy-port)/v1/health > /dev/null && echo "✅ NTFY health OK" || echo "❌ NTFY health failed"

# Debug mode commands
debug-on:
	@echo "🐛 Switching to debug mode..."
	@# Save current Claude config if it exists
	@if [ -f ./.claude/settings.local.json ]; then \
		cp ./.claude/settings.local.json ./.claude/settings.local.json.backup; \
		echo "📋 Backed up existing Claude config to .claude/settings.local.json.backup"; \
	fi
	@# Stop normal server if running
	@docker compose down 2>/dev/null || true
	@echo "🛑 Stopped normal server"
	@# Create debug hooks configuration
	@cp hooks.json ./.claude/settings.local.json
	@echo "🔧 Created debug hooks configuration in .claude/settings.local.json"
	@# Start debug server
	@docker compose -f docker-compose.debug.yml up -d
	@echo "✅ Debug server started on http://localhost:$$(make -s get-debug-port)"
	@echo "🐛 Debug endpoints: http://localhost:$$(make -s get-debug-port)/debug/webhook/"
	@echo "📋 Info page: http://localhost:$$(make -s get-debug-port)/"
	@echo ""
	@echo "🚀 Debug mode is now active!"
	@echo "   All Claude Code hooks will now send requests to the debug server"
	@echo "   Use 'make debug-off' to return to normal mode"

debug-off:
	@echo "🔄 Switching back to normal mode..."
	@# Stop debug server
	@docker compose -f docker-compose.debug.yml down 2>/dev/null || true
	@echo "🛑 Stopped debug server"
	@# Restore original Claude config
	@if [ -f ./.claude/settings.local.json.backup ]; then \
		mv ./.claude/settings.local.json.backup ./.claude/settings.local.json; \
		echo "📋 Restored original Claude config from backup"; \
	else \
		rm -f ./.claude/settings.local.json; \
		echo "📋 Removed debug hooks configuration"; \
	fi
	@# Start normal server
	@docker compose up -d
	@echo "✅ Normal server started"
	@echo "📱 Dashboard: http://localhost:$$(make -s get-server-port)/dashboard"
	@echo "🔗 Webhook: http://localhost:$$(make -s get-server-port)/webhook/"
	@echo ""
	@echo "🚀 Normal mode is now active!"
	@echo "   Claude Code hooks are back to their original configuration"
