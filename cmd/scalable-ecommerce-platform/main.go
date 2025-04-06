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

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/handlers"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/sendGrid"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/stripe"
)

func main() {

	// Load config
	cfg := config.MustLoad()

	// Database setup
	repos, err := repository.New(cfg)

	if err != nil {
		log.Fatalf("❌ Error accessing the database: %v", err)
	}

	// Redis setup
	redisRepo, err := repository.NewRedisRepo(cfg)

	if err != nil {
		log.Fatalf("❌ Error accessing the redis instance: %v", err)
	}

	defer func() {
		if err := repos.Close(); err != nil {
			slog.Error("⚠️ Error closing database connection", slog.String("error", err.Error()))
		} else {
			slog.Info("✅ Database connection closed")
		}
	}()

	jwtKey := []byte(cfg.Security.JWTKey)
	stripeClient := stripe.NewStripeClient(cfg.Stripe.APIKey, cfg.Stripe.WebhookSecret)
	sendGridClient := sendGrid.NewEmailService(cfg.SendGrid.APIKey, cfg.SendGrid.FromEmail, cfg.SendGrid.FromName)
	userService := service.NewUserService(repos.User, redisRepo, jwtKey)
	userHandler := handlers.NewUserHandler(userService)
	productService := service.NewProductService(repos.Product)
	productHandler := handlers.NewProductHandler(productService)
	cartService := service.NewCartService(repos.Cart)
	cartHandler := handlers.NewCartHandler(cartService)
	orderService := service.NewOrderService(repos.Order, repos.Cart, repos.Product)
	orderHandler := handlers.NewOrderHandler(orderService)
	paymentService := service.NewPaymentService(repos.Payment, stripeClient)
	paymentHandler := handlers.NewPaymentHandler(paymentService)
	notificationService := service.NewNotificationService(repos.Notification, repos.User, sendGridClient)
	notificationHandler := handlers.NewNotificationHandler(notificationService)
	authMiddleware := middleware.NewAuthMiddleware(jwtKey)

	slog.Info("storage initialized", slog.String("env", cfg.Env), slog.String("version", "1.0.0"))

	// Setup router
	router := http.NewServeMux()
	router.HandleFunc("POST /api/v1/register", userHandler.Register())
	router.HandleFunc("POST /api/v1/login", userHandler.Login())
	router.HandleFunc("GET /api/v1/profile", authMiddleware.Authenticate(userHandler.Profile()))
	router.HandleFunc("POST /api/v1/products", authMiddleware.Authenticate(productHandler.CreateProduct()))
	router.HandleFunc("GET /api/v1/products/{id}", authMiddleware.Authenticate(productHandler.GetProduct()))
	router.HandleFunc("PUT /api/v1/products/{id}", authMiddleware.Authenticate(productHandler.UpdateProduct()))
	router.HandleFunc("GET /api/v1/products", authMiddleware.Authenticate(productHandler.ListProducts()))
	router.HandleFunc("GET /api/v1/carts", authMiddleware.Authenticate(cartHandler.GetCart()))
	router.HandleFunc("POST /api/v1/carts/items", authMiddleware.Authenticate(cartHandler.AddItem()))
	router.HandleFunc("PUT /api/v1/carts/items", authMiddleware.Authenticate(cartHandler.UpdateQuantity()))
	router.HandleFunc("POST /api/v1/orders", authMiddleware.Authenticate(orderHandler.CreateOrder()))
	router.HandleFunc("GET /api/v1/orders/{id}", authMiddleware.Authenticate(orderHandler.GetOrder()))
	router.HandleFunc("GET /api/v1/orders", authMiddleware.Authenticate(orderHandler.ListOrders()))
	router.HandleFunc("PATCH /api/v1/orders/{id}/status", authMiddleware.Authenticate(orderHandler.UpdateOrderStatus()))
	router.HandleFunc("POST /api/v1/payments", authMiddleware.Authenticate(paymentHandler.CreatePayment()))
	router.HandleFunc("GET /api/v1/payments/{id}", authMiddleware.Authenticate(paymentHandler.GetPayment()))
	router.HandleFunc("GET /api/v1/payments", authMiddleware.Authenticate(paymentHandler.ListPayments()))
	router.HandleFunc("POST /api/v1/payments/webhook", paymentHandler.HandleStripeWebhook())
	router.HandleFunc("POST /api/v1/notifications/email", authMiddleware.Authenticate(notificationHandler.SendEmail()))
	router.HandleFunc("GET /api/v1/notifications", authMiddleware.Authenticate(notificationHandler.ListNotifications()))

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
