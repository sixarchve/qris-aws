package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	Registry = prometheus.NewRegistry()

	appMetrics = promauto.With(Registry)

	httpRequestsTotal = appMetrics.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = appMetrics.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// serviceHealth tracks dependency liveness: 1 = up, 0 = down.
	// Updated on every call to GET /api/health.
	serviceHealth = appMetrics.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "service_health",
			Help: "Dependency health status: 1=up, 0=down",
		},
		[]string{"service"},
	)
)

// RecordServiceHealth sets the health gauge for a named dependency.
// Call with up=true when reachable, up=false when not.
func RecordServiceHealth(service string, up bool) {
	val := 0.0
	if up {
		val = 1.0
	}
	serviceHealth.WithLabelValues(service).Set(val)
}

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.FullPath() == "/metrics" || c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}
