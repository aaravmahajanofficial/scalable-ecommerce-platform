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
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repository"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repository/redis"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/service"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/stripe"
)

func main() {

	// Load config
	cfg := config.MustLoad()

	// Database setup
	postgresInstance, userRepo, productRepo, cartRepo, orderRepo, paymentRepo, err := repository.New(cfg)

	if err != nil {
		log.Fatalf("❌ Error accessing the database: %v", err)
	}

	// Redis setup
	redisRepo, err := redis.NewRedisRepo(cfg)

	if err != nil {
		log.Fatalf("❌ Error accessing the redis instance: %v", err)
	}

	defer func() {
		if err := postgresInstance.Close(); err != nil {
			slog.Error("⚠️ Error closing database connection", slog.String("error", err.Error()))
		} else {
			slog.Info("✅ Database connection closed")
		}
	}()

	jwtKey := []byte("secret-key-123")
	stripeClient := stripe.NewStripeClient(cfg.Stripe.APIKey)
	userService := service.NewUserService(userRepo, redisRepo, jwtKey)
	userHandler := handlers.NewUserHandler(userService)
	productService := service.NewProductService(productRepo)
	productHandler := handlers.NewProductHandler(productService)
	cartService := service.NewCartService(cartRepo)
	cartHandler := handlers.NewCartHandler(cartService)
	orderService := service.NewOrderService(orderRepo)
	orderHandler := handlers.NewOrderHandler(orderService)
	paymentService := service.NewPaymentService(paymentRepo, stripeClient)
	paymentHandler := handlers.NewPaymentService(paymentService)
	authMiddleware := middleware.NewAuthMiddleware(jwtKey)

	slog.Info("storage initialized", slog.String("env", cfg.Env), slog.String("version", "1.0.0"))

	// Setup router
	router := http.NewServeMux()
	router.HandleFunc("POST /api/v1/register", userHandler.Register())
	router.HandleFunc("POST /api/v1/login", userHandler.Login())
	router.HandleFunc("GET /api/v1/profile", authMiddleware.Authenticate(http.HandlerFunc(userHandler.Profile())))
	router.HandleFunc("POST /api/v1/products", authMiddleware.Authenticate(http.HandlerFunc(productHandler.CreateProduct())))
	router.HandleFunc("GET /api/v1/products/{id}", authMiddleware.Authenticate(http.HandlerFunc(productHandler.GetProduct())))
	router.HandleFunc("PUT /api/v1/products/{id}", authMiddleware.Authenticate(http.HandlerFunc(productHandler.UpdateProduct())))
	router.HandleFunc("GET /api/v1/products", authMiddleware.Authenticate(http.HandlerFunc(productHandler.ListProducts())))
	router.HandleFunc("POST /api/v1/carts", authMiddleware.Authenticate(http.HandlerFunc(cartHandler.CreateCart())))
	router.HandleFunc("GET /api/v1/carts/{id}", authMiddleware.Authenticate(http.HandlerFunc(cartHandler.GetCart())))
	router.HandleFunc("POST /api/v1/carts/{id}/items", authMiddleware.Authenticate(http.HandlerFunc(cartHandler.AddItem())))
	router.HandleFunc("PUT /api/v1/carts/{id}/items", authMiddleware.Authenticate(http.HandlerFunc(cartHandler.UpdateQuantity())))
	router.HandleFunc("POST /api/v1/orders", authMiddleware.Authenticate(http.HandlerFunc(orderHandler.CreateOrder())))
	router.HandleFunc("GET /api/v1/orders/{id}", authMiddleware.Authenticate(http.HandlerFunc(orderHandler.GetOrder())))
	router.HandleFunc("GET /api/v1/orders", authMiddleware.Authenticate(http.HandlerFunc(orderHandler.ListOrders())))
	router.HandleFunc("PATCH /api/v1/orders/{id}/status", authMiddleware.Authenticate(http.HandlerFunc(orderHandler.UpdateOrderStatus())))
	router.HandleFunc("POST /api/v1/payments", authMiddleware.Authenticate(http.HandlerFunc(paymentHandler.CreatePayment())))
	router.HandleFunc("GET /api/v1/payments/{id}", authMiddleware.Authenticate(http.HandlerFunc(paymentHandler.GetPayment())))
	router.HandleFunc("GET /api/v1/payments", authMiddleware.Authenticate(http.HandlerFunc(paymentHandler.ListPayments())))
	router.HandleFunc("POST /api/v1/payments/webhook", authMiddleware.Authenticate(http.HandlerFunc(paymentHandler.HandleStripeWebhook())))

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
