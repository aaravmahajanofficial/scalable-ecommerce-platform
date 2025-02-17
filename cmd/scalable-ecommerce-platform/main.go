package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/handlers"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/service"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/storage/postgres"
)

func main() {

	// Load config
	cfg := config.MustLoad()

	// Database setup
	storage, err := postgres.New(cfg)

	if err != nil {
		log.Fatal("❌ Error accessing the database:", err)
	}

	defer func() {
		if err := storage.Close(); err != nil {
			slog.Error("⚠️ Error closing database connection", slog.String("error", err.Error()))
		} else {
			slog.Info("✅ Database connection closed")
		}
	}()

	jwtKey := []byte("secret-key-123")
	userService := service.NewUserService(storage, jwtKey)
	userHandler := handlers.NewUserHandler(userService)

	slog.Info("storage initialized", slog.String("env", cfg.Env), slog.String("version", "1.0.0"))

	// Setup router
	router := http.NewServeMux()
	router.HandleFunc("POST /api/v1/register", userHandler.Register())
	router.HandleFunc("POST /api/v1/login", userHandler.Login())

	// Setup http server
	server := http.Server{
		Addr:    cfg.Addr,
		Handler: router,
	}

	slog.Info("🚀 Server is starting...", slog.String("address", cfg.Addr))

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() { // Starts the HTTP server in a new goroutine so it doesn't block the main thread.

		if err := server.ListenAndServe(); err != nil {
			slog.Error("❌ Failed to start server", slog.String("error", err.Error()))
		}
	}()

	<-done // blocking, until no signal is added to "done" channel, after the some signal is received the code after this point would be executed

	slog.Warn("🛑 Shutdown signal received. Preparing to stop the server...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("⚠️ Server shutdown encountered an issue", slog.String("error", err.Error()))
	} else {
		slog.Info("✅ Server shut down gracefully. All connections closed.")
	}

}
