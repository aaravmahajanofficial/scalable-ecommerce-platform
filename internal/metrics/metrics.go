package metrics

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"code", "method", "path"},
	)
	httpRequestsDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current Number of HTTP requests being processed.",
		},
	)
)

func init() {
	if err := prometheus.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})); err != nil {
		slog.Debug("ProcessCollector registration skipped (likely already registered)",
			slog.String("error", err.Error()))
	}

	if err := prometheus.Register(collectors.NewGoCollector()); err != nil {
		slog.Debug("GoCollector registration skipped (likely already registered)",
			slog.String("error", err.Error()))
	}
}

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

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()
		httpRequestsInFlight.Inc()

		rw := newResponseWriter(w)

		pathPattern := r.URL.Path
		if p := r.PathValue("..."); p != "" {

			pathPattern = r.URL.Path[:len(r.URL.Path)-len(p)] + "{...}"

		} else if id := r.PathValue("id"); id != "" {

			pathPattern = r.URL.Path[:len(r.URL.Path)-len(id)] + "{id}"

		}

		defer func() {

			duration := time.Since(start)
			statusCodeStr := strconv.Itoa(rw.statusCode)

			httpRequestsTotal.WithLabelValues(statusCodeStr, r.Method, pathPattern).Inc()
			httpRequestsDuration.WithLabelValues(r.Method, pathPattern).Observe(duration.Seconds())
			httpRequestsInFlight.Dec()

		}()

		next.ServeHTTP(rw, r)

	})
}

// http.Handler for the Prometheus /metrics endpoint
func Handler() http.Handler {

	return promhttp.Handler()
}
