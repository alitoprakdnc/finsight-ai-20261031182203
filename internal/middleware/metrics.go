package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds the Prometheus collectors used by the service.
type Metrics struct {
	requestCount    *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	inFlight        *prometheus.GaugeVec
}

// singleton instance and once guard to avoid double registration.
var (
	once          sync.Once
	defaultMetric *Metrics
)

// RegisterMetrics registers the /metrics endpoint on the provided router and
// returns a Gin middleware that records request metrics.
// It is safe to call multiple times; the collectors are registered only once.
func RegisterMetrics(r *gin.Engine) gin.HandlerFunc {
	once.Do(func() {
		defaultMetric = newMetrics()
		// Expose Prometheus metrics endpoint.
		r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	})
	return defaultMetric.handler()
}

// newMetrics creates the Prometheus collectors, registers them and returns a *Metrics.
func newMetrics() *Metrics {
	m := &Metrics{
		requestCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests processed, partitioned by method, handler and status code.",
			},
			[]string{"method", "handler", "code"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "Histogram of request processing time in seconds, partitioned by method, handler and status code.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "handler", "code"},
		),
		inFlight: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "http_in_flight_requests",
				Help: "Current number of in‑flight HTTP requests, partitioned by method and handler.",
			},
			[]string{"method", "handler"},
		),
	}

	// Register the collectors. MustRegister will panic on duplicate registration,
	// which is fine because we guard with sync.Once.
	prometheus.MustRegister(m.requestCount, m.requestDuration, m.inFlight)

	return m
}

// handler returns a Gin middleware that instruments each request.
func (m *Metrics) handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Resolve the route pattern; fallback to "unknown" for non‑matched routes.
		handler := c.FullPath()
		if handler == "" {
			handler = "unknown"
		}
		method := c.Request.Method

		// Increment in‑flight gauge.
		m.inFlight.WithLabelValues(method, handler).Inc()
		start := time.Now()

		// Process request.
		c.Next()

		// After request processing.
		duration := time.Since(start).Seconds()
		status := c.Writer.Status()
		codeLabel := fmt.Sprintf("%d", status)

		// Record metrics.
		m.requestCount.WithLabelValues(method, handler, codeLabel).Inc()
		m.requestDuration.WithLabelValues(method, handler, codeLabel).Observe(duration)

		// Decrement in‑flight gauge.
		m.inFlight.WithLabelValues(method, handler).Dec()
	}
}

// GetDefault returns the singleton Metrics instance.
// Useful for tests that need direct access to the collectors.
func GetDefault() *Metrics {
	if defaultMetric == nil {
		// Lazily initialise if RegisterMetrics hasn't been called yet.
		once.Do(func() {
			defaultMetric = newMetrics()
		})
	}
	return defaultMetric
}