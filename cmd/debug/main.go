package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpAdapter "github.com/dan/claude-control/internal/adapters/http"
	"github.com/gorilla/mux"
)

// Config holds debug server configuration
type DebugConfig struct {
	ServerPort string `json:"server_port"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *DebugConfig {
	return &DebugConfig{
		ServerPort: getEnv("DEBUG_PORT", "8080"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	log.Println("🐛 Starting Claude Code Debug Server...")
	log.Println("   This server only provides webhook debugging endpoints")
	log.Println("   No database, no task processing, just request logging")

	// Load configuration
	config := LoadConfig()
	log.Printf("Configuration loaded: Debug server will run on port %s", config.ServerPort)

	// Initialize only the test debug handler
	testDebugHandler := httpAdapter.NewTestDebugHandler()
	log.Println("✅ Debug handler initialized")

	// Setup routes
	router := mux.NewRouter()

	// Register only debug routes
	testDebugHandler.RegisterRoutes(router)
	log.Println("✅ Debug webhook routes registered")

	// Add a simple health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy", "service": "claude-debug-server"}`))
	}).Methods("GET")

	// Add a root endpoint that explains what this server does
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Claude Code Debug Server</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; line-height: 1.6; }
        .endpoint { background: #f4f4f4; padding: 10px; margin: 10px 0; border-radius: 4px; }
        code { background: #e8e8e8; padding: 2px 4px; border-radius: 2px; }
    </style>
</head>
<body>
    <h1>🐛 Claude Code Debug Server</h1>
    <p>This server provides debug endpoints for inspecting Claude Code webhook requests.</p>
    
    <h2>Available Webhook Endpoints:</h2>
    <div class="endpoint"><code>POST /webhook/pre-tool-use</code></div>
    <div class="endpoint"><code>POST /webhook/post-tool-use</code></div>
    <div class="endpoint"><code>POST /webhook/notification</code></div>
    <div class="endpoint"><code>POST /webhook/user-prompt-submit</code></div>
    <div class="endpoint"><code>POST /webhook/stop</code></div>
    <div class="endpoint"><code>POST /webhook/subagent-stop</code></div>
    <div class="endpoint"><code>POST /webhook/pre-compact</code></div>
    <div class="endpoint"><code>POST /webhook/{hookType}</code> (generic)</div>
    
    <h2>Debug Endpoints (legacy):</h2>
    <div class="endpoint"><code>POST /debug/webhook/pre-tool-use</code></div>
    <div class="endpoint"><code>POST /debug/webhook/post-tool-use</code></div>
    <div class="endpoint"><code>POST /debug/webhook/notification</code></div>
    <div class="endpoint"><code>POST /debug/webhook/user-prompt-submit</code></div>
    <div class="endpoint"><code>POST /debug/webhook/stop</code></div>
    <div class="endpoint"><code>POST /debug/webhook/subagent-stop</code></div>
    <div class="endpoint"><code>POST /debug/webhook/pre-compact</code></div>
    <div class="endpoint"><code>POST /debug/webhook/{hookType}</code> (generic)</div>
    
    <h2>Other Endpoints:</h2>
    <div class="endpoint"><code>GET /health</code> - Health check</div>
    
    <p>See the server logs for formatted webhook request details when endpoints are called.</p>
    <p>For usage instructions, see <code>DEBUG_WEBHOOK_USAGE.md</code></p>
</body>
</html>
		`))
	}).Methods("GET")

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + config.ServerPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second, // Shorter timeout for debug server
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("🚀 Debug server starting on http://localhost:%s", config.ServerPort)
		log.Printf("📋 Info page: http://localhost:%s/", config.ServerPort)
		log.Printf("💚 Health check: http://localhost:%s/health", config.ServerPort)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start debug server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down debug server...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown server gracefully
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Debug server forced to shutdown: %v", err)
	}

	log.Println("✅ Debug server shutdown complete")
}