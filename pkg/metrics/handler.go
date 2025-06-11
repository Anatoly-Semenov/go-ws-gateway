package metrics

import (
	"net/http"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type MetricsHandler struct {
	metrics *Metrics
	logger  *zap.Logger
}

func NewMetricsHandler(metrics *Metrics, logger *zap.Logger) *MetricsHandler {
	handler := &MetricsHandler{
		metrics: metrics,
		logger:  logger,
	}

	go handler.collectSystemMetrics()

	return handler
}

func (h *MetricsHandler) Handler() http.Handler {
	return promhttp.Handler()
}

func (h *MetricsHandler) collectSystemMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		
		h.metrics.System.GoroutineCount.Set(float64(runtime.NumGoroutine()))

	}
}

func (h *MetricsHandler) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {
	h.metrics.Http.RequestsTotal.WithLabelValues(method, path).Inc()
	h.metrics.Http.ResponseStatusCode.WithLabelValues(http.StatusText(statusCode)).Inc()
	h.metrics.Http.RequestDuration.WithLabelValues(path).Observe(duration.Seconds())
}
