package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpAdapter "github.com/dan/claude-control/internal/adapters/http"
	"github.com/dan/claude-control/internal/adapters/ntfy"
	"github.com/dan/claude-control/internal/adapters/postgres"
	"github.com/dan/claude-control/internal/adapters/response"
	"github.com/dan/claude-control/internal/core/domain"
	"github.com/dan/claude-control/internal/core/ports"
	"github.com/dan/claude-control/internal/core/services"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Config holds application configuration
type Config struct {
	ServerPort    string `json:"server_port"`
	DatabaseURL   string `json:"database_url"`
	NTFYServerURL string `json:"ntfy_server_url"`
	NTFYTopic     string `json:"ntfy_topic"`
	WebDomain     string `json:"web_domain"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgresql://claude_user:claude_password@localhost:5432/claude_control?sslmode=disable"),
		NTFYServerURL: getEnv("NTFY_SERVER_URL", "http://localhost:80"),
		NTFYTopic:     getEnv("NTFY_TOPIC", "claude-notifications"),
		WebDomain:     getEnv("WEB_DOMAIN", "localhost:8080"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	log.Println("🤖 Starting Claude Control Server...")

	// Load configuration
	config := LoadConfig()
	log.Printf("Configuration loaded: Server will run on port %s", config.ServerPort)

	// Initialize database connection
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("✅ Database connection established")

	// Initialize repositories
	taskRepo := postgres.NewTaskRepository(db)
	historyRepo := postgres.NewTaskHistoryRepository(db)
	log.Println("✅ Repository adapters initialized")

	// Initialize notification sender
	notificationConfig := &ports.NotificationConfig{
		ServerURL: config.NTFYServerURL,
		Topic:     config.NTFYTopic,
	}
	notificationSender := ntfy.NewNotificationSender(notificationConfig)

	// Verify notification service
	if err := notificationSender.Verify(ctx); err != nil {
		log.Printf("⚠️ Warning: NTFY service verification failed: %v", err)
		log.Println("   Notifications may not work properly")
	} else {
		log.Println("✅ NTFY notification service verified")
	}

	// Initialize hook response builder
	responseBuilder := response.NewHookResponseBuilder()
	log.Println("✅ Hook response builder initialized")

	// Initialize task service
	taskServiceConfig := &services.TaskServiceConfig{
		WebDomain: config.WebDomain,
		AutoNotifyHookTypes: []domain.HookType{
			domain.HookTypePreToolUse,
			domain.HookTypeUserPromptSubmit,
			// Note: Stop and Notification webhooks are now non-blocking
			// They create tasks for logging but don't require user notifications
		},
	}
	taskService := services.NewTaskService(
		taskRepo,
		historyRepo,
		notificationSender,
		responseBuilder,
		taskServiceConfig,
	)
	log.Println("✅ Task service initialized")

	// Initialize HTTP handlers
	webhookHandler := httpAdapter.NewWebhookHandler(taskService)
	webHandler := httpAdapter.NewWebHandler(taskService, webhookHandler)
	testDebugHandler := httpAdapter.NewTestDebugHandler()
	log.Println("✅ HTTP handlers initialized")

	// Setup routes
	router := mux.NewRouter()

	// Register webhook routes
	webhookHandler.RegisterRoutes(router)
	log.Println("✅ Webhook routes registered")

	// Register web interface routes
	webHandler.RegisterRoutes(router)
	log.Println("✅ Web interface routes registered")

	// Register test debug routes
	testDebugHandler.RegisterRoutes(router)
	log.Println("✅ Test debug routes registered")

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + config.ServerPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 6 * time.Minute, // Allow time for blocking webhook decisions (5min + buffer)
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("🚀 Server starting on http://localhost:%s", config.ServerPort)
		log.Printf("📱 Dashboard: http://localhost:%s/dashboard", config.ServerPort)
		log.Printf("🔗 Webhook endpoint: http://localhost:%s/webhook/", config.ServerPort)
		log.Printf("🐛 Debug webhook endpoint: http://localhost:%s/debug/webhook/", config.ServerPort)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown server gracefully
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server shutdown complete")
}
