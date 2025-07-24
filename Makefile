# Claude Control Server Makefile
# Provides convenient commands for Docker Compose management

.PHONY: help up down restart logs build clean status shell-server shell-db test db-clean-pending db-clean-all-tasks db-status db-recent-tasks db-shell health

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

# Main commands
up:
	docker compose up -d
	@echo "✅ All services started"
	@echo "📱 Dashboard: http://localhost:8080/dashboard"
	@echo "🔗 Webhook: http://localhost:8080/webhook/"

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

# Utility commands
shell-server:
	docker compose exec claude-control /bin/sh

shell-db:
	docker compose exec postgres psql -U claude_user -d claude_control

test:
	@echo "🧪 Testing webhook endpoint..."
	curl -X POST http://localhost:8080/webhook/pre-tool-use \
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
	@curl -s http://localhost:8080/health > /dev/null && echo "✅ Server health OK" || echo "❌ Server health failed"
	@curl -s http://localhost:80/v1/health > /dev/null && echo "✅ NTFY health OK" || echo "❌ NTFY health failed"