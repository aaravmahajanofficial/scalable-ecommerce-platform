package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/aaravmahajanofficial/scalable-ecommerce-platform/docs"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/handlers"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/cache"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/health"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/metrics"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/sendGrid"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/stripe"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
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

// Creates and Register the Jaeger exporter and OTel TracerProvider
func initTracer(cfg *config.Config) (func(ctx context.Context) error, error) {

	ctx := context.Background()

	exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(cfg.OTel.ExporterEndpoint), otlptracehttp.WithURLPath("/v1/traces"), otlptracehttp.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.OTel.ServiceName),
			semconv.ServiceVersion("1.0.0"),
			semconv.DeploymentEnvironmentName(cfg.Env),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	samplingRatio := cfg.OTel.SamplerRatio
	if samplingRatio <= 0 || samplingRatio > 1 {
		samplingRatio = 1.0
	}

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(samplingRatio))

	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter), sdktrace.WithResource(res), sdktrace.WithSampler(sampler))
	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	slog.Info("OpenTelemetry Tracer initialized",
		slog.String("service_name", cfg.OTel.ServiceName),
		slog.String("exporter_endpoint", cfg.OTel.ExporterEndpoint),
		slog.Float64("sampling_ratio", samplingRatio),
	)

	return func(ctx context.Context) error {
		shutdown, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		return tp.Shutdown(shutdown)
	}, nil
}

func main() {

	// Logger setup
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Load config
	cfg := config.MustLoad()

	tracerShutdown, err := initTracer(cfg)
	if err != nil {
		slog.Error("❌ Failed to initialize OpenTelemetry Tracer", "error", err.Error())
		os.Exit(1)
	}
	defer func() {
		slog.Info("Shutting down tracer...")
		if err := tracerShutdown(context.Background()); err != nil {
			slog.Error("⚠️ Error shutting down tracer", "error", err)
		} else {
			slog.Info("✅ Tracer shut down successfully.")
		}
	}()

	// Swagger setup
	swaggerHost := cfg.Addr
	if swaggerHost == "" {
		swaggerHost = "0.0.0.0:8085"
		slog.Warn("Server address not found in config (cfg.Addr), defaulting Swagger host to " + swaggerHost)
	}

	// --- Redis Client Initialization ---
	redisClient, err := repository.NewRedisClient(cfg)
	if err != nil {
		slog.Error("❌ Failed to initialize Redis client", "error", err.Error())
		os.Exit(1)
	}
	// Defer closing the Redis client connection
	defer func() {
		slog.Info("Closing Redis connection...")
		if err := redisClient.Close(); err != nil {
			slog.Error("⚠️ Error closing Redis connection", slog.String("error", err.Error()))
		} else {
			slog.Info("✅ Redis connection closed")
		}
	}()

	// --- Cache Initialization ---
	redisCache := cache.NewRedisCache(redisClient, &cfg.Cache)
	slog.Info("Cache Initialized", slog.String("type", "redis"), slog.String("defaultTTL", cfg.Cache.DefaultTTL.String()))

	// --- Rate Limiter Initialization ---
	rateLimiter := repository.NewRateLimitRepo(redisClient, cfg)
	slog.Info("Rate Limiter Initialized", slog.String("type", "redis"))

	// --- Database and Repositories Initialization ---
	repos, err := repository.New(cfg, redisClient, redisCache, rateLimiter)
	if err != nil {
		slog.Error("❌ Error initializing repositories", "error", err.Error())
		os.Exit(1)
	}
	// Defer closing the DB connection
	defer func() {
		slog.Info("Closing repository connections (DB, Redis)...")
		if err := repos.Close(); err != nil {
			slog.Error("⚠️ Error closing repository connections", slog.String("error", err.Error()))
		} else {
			slog.Info("✅ Repository connections closed")
		}
	}()

	jwtKey := []byte(cfg.Security.JWTKey)
	stripeClient := stripe.NewStripeClient(cfg.Stripe.APIKey, cfg.Stripe.WebhookSecret)
	sendGridClient := sendGrid.NewEmailService(cfg.SendGrid.APIKey, cfg.SendGrid.FromEmail, cfg.SendGrid.FromName)

	// Service Init
	userService := service.NewUserService(repos.User, repos.RateLimiter, jwtKey)
	productService := service.NewProductService(repos.Product)
	cartService := service.NewCartService(repos.Cart)
	orderService := service.NewOrderService(repos.Order, repos.Cart, repos.Product)
	paymentService := service.NewPaymentService(repos.Payment, stripeClient)
	notificationService := service.NewNotificationService(repos.Notification, repos.User, sendGridClient)

	// Handler Init
	userHandler := handlers.NewUserHandler(userService)
	productHandler := handlers.NewProductHandler(productService)
	cartHandler := handlers.NewCartHandler(cartService)
	orderHandler := handlers.NewOrderHandler(orderService)
	paymentHandler := handlers.NewPaymentHandler(paymentService)
	notificationHandler := handlers.NewNotificationHandler(notificationService)

	// Middleware Init
	authMiddleware := middleware.NewAuthMiddleware(jwtKey)

	slog.Info("Storage Initialized", slog.String("env", cfg.Env), slog.String("version", "1.0.0"))

	healthEndpoints := &health.HealthEndpoint{
		DB:           repos.DB,
		RedisClient:  repos.RedisClient,
		StripeClient: &stripeClient,
	}

	readinessHandler, err := health.NewReadinessHandler(cfg, healthEndpoints)
	if err != nil {
		slog.Error("❌ Failed to initialize readiness checker", "error", err.Error())
		os.Exit(1)
	}

	livenessHandler := health.NewLivenessHandler()

	slog.Info("✅ Health checks initialized")

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

	// Liveness check endpoint
	mainMux.Handle("/livez", livenessHandler)
	slog.Info("⚕️ Liveness probe available", slog.String("path", "/livez"))

	// Readiness check endpoint
	mainMux.Handle("/readyz", readinessHandler)
	slog.Info("⚕️ Readiness probe available", slog.String("path", "/readyz"))

	// Swagger UI enpoint
	mainMux.Handle("/swagger/", httpSwagger.WrapHandler)
	slog.Info("Swagger UI available at http://" + swaggerHost + "/swagger/index.html")

	var apiHandler http.Handler = apiMux // raw router as base handler

	// Middleware chaining -> Reverse order of execution, 
	apiHandler = middleware.Logging(apiHandler) // Log all info
	apiHandler = metrics.Middleware(apiHandler)
	apiHandler = otelhttp.NewHandler(apiHandler, cfg.OTel.ServiceName) //  Wraps actual business logic

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
	slog.Info("📊 Metrics available", slog.String("path", "/metrics"))

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() { // Starts the HTTP server in a new goroutine so it doesn't block the main thread.

		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("❌ Server failed to start", "error", err.Error())
			close(done)
		}
	}()

	slog.Info("✅ Server started successfully")
	<-done // blocking, until no signal is added to "done" channel, after the some signal is received the code after this point would be executed

	// Graceful shutdown
	slog.Info("⏳ Server shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("⚠️ Server shutdown failed", "error", err)
	} else {
		slog.Info("✅ Server shutdown complete")
	}

}
