package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type logContextKey string

const loggerKey = logContextKey("logger")

// wrapper around http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// main middleware
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()

		// Correlation ID
		correlationID := r.Header.Get("X-Request-ID")
		if correlationID == "" {
			correlationID = uuid.NewString()
		}

		w.Header().Set("X-Request-ID", correlationID)

		// Request-scoper logger, every log line would contain these fields
		requestLogger := slog.Default().With(
			slog.String("correlation_id", correlationID),
			slog.String("http_method", r.Method),
			slog.String("http_path", r.URL.Path),
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
		)

		// Incoming request log
		requestLogger.Info("Incoming request")

		ctx := context.WithValue(r.Context(), loggerKey, requestLogger)

		rw := newResponseWriter(w)

		next.ServeHTTP(w, r.WithContext(ctx))

		// log the completed request
		requestLogger.Info("Request Completed", slog.Int("http_status", rw.statusCode), slog.Duration("duration", time.Since(start)))

	})
}

func LoggerFromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}

	return slog.Default()
}
