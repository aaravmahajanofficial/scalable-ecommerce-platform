package main

import (
	"log"
	"log/slog"
	"net/http"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
)

func main() {

	// load config

	cfg := config.MustLoad()

	// database setup

	// setup router

	router := http.NewServeMux()

	router.HandleFunc("POST /api/v1/register", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Sign up to get started..."))
	})
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

	slog.Info("Started server at", slog.String("address", cfg.Addr))

	err := server.ListenAndServe()

	if err != nil {
		log.Fatalf("error starting the server %s", err)
	}

}
