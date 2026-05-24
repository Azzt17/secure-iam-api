package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Sensor 1: Menghitung total request (Berguna untuk deteksi DDoS)
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total jumlah HTTP request yang masuk",
		},
		[]string{"method", "path", "status"},
	)

	// Sensor 2: Mengukur durasi eksekusi (Berguna untuk deteksi Slowloris / performa DB anjlok)
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Durasi HTTP request dalam detik",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

// Metrics adalah middleware untuk merekam HTTP request
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rec, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(rec.statusCode)

		// Mencegah Cardinality Explosion dgn hanya rekam path saja
		path := r.URL.Path

		if path == "/metrics" {
			return
		}

		httpRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}
