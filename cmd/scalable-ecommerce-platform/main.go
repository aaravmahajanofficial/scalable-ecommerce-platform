package main

import (
	"context"
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

	// Logger setup
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Load config
	cfg := config.MustLoad()

	// Database setup
	repos, err := repository.New(cfg)
	if err != nil {
		slog.Error("❌ Error accessing the database", "error", err.Error())
		os.Exit(1)
	}

	// Redis setup
	redisRepo, err := repository.NewRedisRepo(cfg)
	if err != nil {
		slog.Error("❌ Error accessing the redis instance", "error", err.Error())
		os.Exit(1)
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
	routerMux := http.NewServeMux()
	routerMux.HandleFunc("POST /api/v1/users/register", userHandler.Register())
	routerMux.HandleFunc("POST /api/v1/users/login", userHandler.Login())
	routerMux.HandleFunc("GET /api/v1/users/profile", authMiddleware.Authenticate(userHandler.Profile()))
	routerMux.HandleFunc("POST /api/v1/products", authMiddleware.Authenticate(productHandler.CreateProduct()))
	routerMux.HandleFunc("GET /api/v1/products/{id}", authMiddleware.Authenticate(productHandler.GetProduct()))
	routerMux.HandleFunc("PUT /api/v1/products/{id}", authMiddleware.Authenticate(productHandler.UpdateProduct()))
	routerMux.HandleFunc("GET /api/v1/products", authMiddleware.Authenticate(productHandler.ListProducts()))
	routerMux.HandleFunc("GET /api/v1/carts", authMiddleware.Authenticate(cartHandler.GetCart()))
	routerMux.HandleFunc("POST /api/v1/carts/items", authMiddleware.Authenticate(cartHandler.AddItem()))
	routerMux.HandleFunc("PUT /api/v1/carts/items", authMiddleware.Authenticate(cartHandler.UpdateQuantity()))
	routerMux.HandleFunc("POST /api/v1/orders", authMiddleware.Authenticate(orderHandler.CreateOrder()))
	routerMux.HandleFunc("GET /api/v1/orders/{id}", authMiddleware.Authenticate(orderHandler.GetOrder()))
	routerMux.HandleFunc("GET /api/v1/orders", authMiddleware.Authenticate(orderHandler.ListOrders()))
	routerMux.HandleFunc("PATCH /api/v1/orders/{id}/status", authMiddleware.Authenticate(orderHandler.UpdateOrderStatus()))
	routerMux.HandleFunc("POST /api/v1/payments", authMiddleware.Authenticate(paymentHandler.CreatePayment()))
	routerMux.HandleFunc("GET /api/v1/payments/{id}", authMiddleware.Authenticate(paymentHandler.GetPayment()))
	routerMux.HandleFunc("GET /api/v1/payments", authMiddleware.Authenticate(paymentHandler.ListPayments()))
	routerMux.HandleFunc("POST /api/v1/payments/webhook", paymentHandler.HandleStripeWebhook())
	routerMux.HandleFunc("POST /api/v1/notifications/email", authMiddleware.Authenticate(notificationHandler.SendEmail()))
	routerMux.HandleFunc("GET /api/v1/notifications", authMiddleware.Authenticate(notificationHandler.ListNotifications()))

	// Middleware chaining
	var handler http.Handler = routerMux
	handler = middleware.Logging(handler)

	// Setup http server
	server := http.Server{
		Addr:    cfg.Addr,
		Handler: handler,
	}

	slog.Info("🚀 Server is starting...", slog.String("address", cfg.Addr))

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() { // Starts the HTTP server in a new goroutine so it doesn't block the main thread.

		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("❌ Failed to start server", slog.Any("error", err.Error()))
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
