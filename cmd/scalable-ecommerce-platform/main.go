package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/aaravmahajanofficial/scalable-ecommerce-platform/docs"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/handlers"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/metrics"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/sendGrid"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/stripe"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title           Scalable E-commerce Platform API
// @version         1.0
// @description     This is the API server for the Scalable E-commerce Platform. It provides endpoints for managing users, products, carts, orders, payments, and notifications.
// @termsOfService  http://swagger.io/terms/

// @contact.name   Aarav Mahajan
// @contact.url    https://github.com/aaravmahajanofficial
// @contact.email  aaravmahajan2003@gmail.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      0.0.0.0:8085
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token. Example: "Bearer {token}"
func main() {

	// Logger setup
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Load config
	cfg := config.MustLoad()

	// Swagger setup
	swaggerHost := cfg.Addr
	if swaggerHost == "" {
		swaggerHost = "0.0.0.0:8085"
		slog.Warn("Server address not found in config (cfg.Addr), defaulting Swagger host to " + swaggerHost)
	}

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

	// Setup router for handling api routes only
	apiMux := http.NewServeMux()

	apiMux.HandleFunc("POST /api/v1/users/register", userHandler.Register())
	apiMux.HandleFunc("POST /api/v1/users/login", userHandler.Login())
	apiMux.HandleFunc("GET /api/v1/users/profile", authMiddleware.Authenticate(userHandler.Profile()))
	apiMux.HandleFunc("POST /api/v1/products", authMiddleware.Authenticate(productHandler.CreateProduct()))
	apiMux.HandleFunc("GET /api/v1/products/{id}", authMiddleware.Authenticate(productHandler.GetProduct()))
	apiMux.HandleFunc("PUT /api/v1/products/{id}", authMiddleware.Authenticate(productHandler.UpdateProduct()))
	apiMux.HandleFunc("GET /api/v1/products", authMiddleware.Authenticate(productHandler.ListProducts()))
	apiMux.HandleFunc("GET /api/v1/carts", authMiddleware.Authenticate(cartHandler.GetCart()))
	apiMux.HandleFunc("POST /api/v1/carts/items", authMiddleware.Authenticate(cartHandler.AddItem()))
	apiMux.HandleFunc("PUT /api/v1/carts/items", authMiddleware.Authenticate(cartHandler.UpdateQuantity()))
	apiMux.HandleFunc("POST /api/v1/orders", authMiddleware.Authenticate(orderHandler.CreateOrder()))
	apiMux.HandleFunc("GET /api/v1/orders/{id}", authMiddleware.Authenticate(orderHandler.GetOrder()))
	apiMux.HandleFunc("GET /api/v1/orders", authMiddleware.Authenticate(orderHandler.ListOrders()))
	apiMux.HandleFunc("PATCH /api/v1/orders/{id}/status", authMiddleware.Authenticate(orderHandler.UpdateOrderStatus()))
	apiMux.HandleFunc("POST /api/v1/payments", authMiddleware.Authenticate(paymentHandler.CreatePayment()))
	apiMux.HandleFunc("GET /api/v1/payments/{id}", authMiddleware.Authenticate(paymentHandler.GetPayment()))
	apiMux.HandleFunc("GET /api/v1/payments", authMiddleware.Authenticate(paymentHandler.ListPayments()))
	apiMux.HandleFunc("POST /api/v1/payments/webhook", authMiddleware.Authenticate(paymentHandler.HandleStripeWebhook()))
	apiMux.HandleFunc("POST /api/v1/notifications/email", authMiddleware.Authenticate(notificationHandler.SendEmail()))
	apiMux.HandleFunc("GET /api/v1/notifications", authMiddleware.Authenticate(notificationHandler.ListNotifications()))

	// Main router
	mainMux := http.NewServeMux()

	// Metrics handler
	mainMux.Handle("/metrics", metrics.Handler())

	// Swagger UI enpoint handler
	mainMux.Handle("/swagger/", httpSwagger.WrapHandler)
	slog.Info("Swagger UI available at http://" + swaggerHost + "/swagger/index.html")

	// Middleware chaining
	var apiHandler http.Handler = apiMux // raw router as base handler
	apiHandler = metrics.Middleware(apiHandler)
	apiHandler = middleware.Logging(apiHandler)

	mainMux.Handle("/api/v1/", apiHandler)

	// Setup http server
	server := http.Server{
		Addr:         cfg.Addr,
		Handler:      mainMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	slog.Info("🚀 Server is starting...", slog.String("address", cfg.Addr))
	slog.Info("📊 Metrics available at http://" + cfg.Addr + "/metrics") // Log metrics endpoint

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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("⚠️ Server shutdown encountered an issue", slog.String("error", err.Error()))
	} else {
		slog.Info("✅ Server shut down gracefully. All connections closed.")
	}

}
