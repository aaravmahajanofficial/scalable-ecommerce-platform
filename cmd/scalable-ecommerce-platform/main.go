package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	auth "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/http/handlers/user"
)

func main() {

	// load config

	cfg := config.MustLoad()

	// database setup

	// setup router

	router := http.NewServeMux()

	router.HandleFunc("POST /api/v1/register", auth.Register())
	router.HandleFunc("POST /api/v1/login", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Login to get started..."))
	})
	router.HandleFunc("GET /api/v1/profile", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Here is your profile..."))
	})
	router.HandleFunc("PUT /api/v1/profile", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Profile is updated..."))
	})

	// setup server

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

	<-done

	slog.Warn("🛑 Shutdown signal received. Preparing to stop the server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("⚠️ Server shutdown encountered an issue", slog.String("error", err.Error()))
	} else {
		slog.Info("✅ Server shut down gracefully. All connections closed.")
	}

}
