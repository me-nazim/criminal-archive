// Package metrics exposes a /metrics endpoint with the standard Go +
// process collectors plus a small set of HTTP-request labels. This is
// the v1 surface for Prometheus scraping; we explicitly avoid OTel
// until a debugging session demands distributed tracing.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Number of HTTP requests served, labelled by method, route, status.",
		},
		[]string{"method", "route", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests, labelled by method and route.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)
)

// Registry is the singleton registry the /metrics handler reads from.
// Exported so tests can register additional collectors against it.
var Registry = prometheus.NewRegistry()

func init() {
	Registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		httpRequestsTotal,
		httpRequestDuration,
	)
}

// Handler returns the /metrics http.Handler.
func Handler() http.Handler {
	return promhttp.HandlerFor(Registry, promhttp.HandlerOpts{Registry: Registry})
}

// Middleware records request_total + duration histograms. It must be
// installed AFTER chi's routing has happened so we can read
// chi.RouteContext for the route pattern (otherwise we get noisy
// per-URL labels).
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		dur := time.Since(start).Seconds()

		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}
		status := strconv.Itoa(ww.Status())
		httpRequestsTotal.WithLabelValues(r.Method, route, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, route).Observe(dur)
	})
}
