// Package metrics defines the Prometheus instrumentation for the turl service.
//
// Metrics are registered on a private registry (not the global default) so the
// service controls exactly what is exposed. Collectors and observation helpers
// are package-level: callers record events through the helper functions without
// having to thread a metrics object through every layer.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// namespace prefixes every metric name, e.g. turl_http_requests_total.
const namespace = "turl"

// Cache layer labels.
const (
	CacheLayerLocal       = "local"
	CacheLayerDistributed = "distributed"
)

var registry = prometheus.NewRegistry()

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests, partitioned by method, route, and status.",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request latency in seconds, partitioned by method and route.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	cacheRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "cache",
			Name:      "requests_total",
			Help:      "Total number of cache lookups, partitioned by layer and result (hit/miss).",
		},
		[]string{"layer", "result"},
	)

	shortenCreatedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "shorten_created_total",
			Help:      "Total number of short links successfully created.",
		},
	)
)

func init() {
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		httpRequestsTotal,
		httpRequestDuration,
		cacheRequestsTotal,
		shortenCreatedTotal,
	)
}

// Handler returns an HTTP handler that exposes the registered metrics in the
// Prometheus text format. Mount it at /metrics on the admin server.
func Handler() http.Handler {
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}

// ObserveHTTPRequest records a single served HTTP request: it increments the
// request counter and observes its latency.
func ObserveHTTPRequest(method, path, status string, seconds float64) {
	httpRequestsTotal.WithLabelValues(method, path, status).Inc()
	httpRequestDuration.WithLabelValues(method, path).Observe(seconds)
}

// RecordCacheResult records the outcome of a cache lookup for the given layer
// (CacheLayerLocal or CacheLayerDistributed).
func RecordCacheResult(layer string, hit bool) {
	result := "miss"
	if hit {
		result = "hit"
	}

	cacheRequestsTotal.WithLabelValues(layer, result).Inc()
}

// IncShortenCreated increments the count of successfully created short links.
func IncShortenCreated() {
	shortenCreatedTotal.Inc()
}
