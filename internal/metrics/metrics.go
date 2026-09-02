package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus collectors
type Metrics struct {
	RequestsTotal            *prometheus.CounterVec
	QueueDepth               prometheus.Gauge
	InFlight                 prometheus.Gauge
	QueueWaitSeconds         prometheus.Histogram
	TimeToFirstTokenSeconds  prometheus.Histogram
	TimePerOutputTokenSeconds prometheus.Histogram
	GenerationSeconds        prometheus.Histogram
	TokensTotal              *prometheus.CounterVec
	CacheHitsTotal           prometheus.Counter
	CacheMissesTotal         prometheus.Counter
	BatchSize                prometheus.Histogram
	RejectedTotal            *prometheus.CounterVec
	StreamingConnections     prometheus.Gauge
	// Backend metrics
	BackendHealth            *prometheus.GaugeVec
	BackendRequestsTotal     *prometheus.CounterVec
	BackendLatencySeconds    *prometheus.HistogramVec
	BackendInFlight          *prometheus.GaugeVec
	BackendCircuitOpen       *prometheus.GaugeVec
}

// New creates and registers all metrics
func New() *Metrics {
	return &Metrics{
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_requests_total",
				Help: "Total number of requests by model and status",
			},
			[]string{"model", "status"},
		),
		QueueDepth: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "gateway_queue_depth",
				Help: "Current number of requests in the admission queue",
			},
		),
		InFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "gateway_in_flight",
				Help: "Current number of requests being processed",
			},
		),
		QueueWaitSeconds: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "gateway_queue_wait_seconds",
				Help:    "Time spent waiting in the admission queue",
				Buckets: prometheus.DefBuckets,
			},
		),
		TimeToFirstTokenSeconds: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "gateway_time_to_first_token_seconds",
				Help:    "Time from request admission to first token generated (TTFT)",
				Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
			},
		),
		TimePerOutputTokenSeconds: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "gateway_time_per_output_token_seconds",
				Help:    "Average time per output token (TPOT) for streaming responses",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
			},
		),
		GenerationSeconds: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "gateway_generation_seconds",
				Help:    "Total time spent generating tokens",
				Buckets: prometheus.DefBuckets,
			},
		),
		TokensTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_tokens_total",
				Help: "Total tokens processed by type (prompt or completion)",
			},
			[]string{"type"},
		),
		CacheHitsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "gateway_cache_hits_total",
				Help: "Total number of cache hits",
			},
		),
		CacheMissesTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "gateway_cache_misses_total",
				Help: "Total number of cache misses",
			},
		),
		BatchSize: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "gateway_batch_size",
				Help:    "Size of batches dispatched for embeddings",
				Buckets: []float64{1, 2, 4, 8, 16, 32, 64},
			},
		),
		RejectedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_rejected_total",
				Help: "Total number of rejected requests by reason",
			},
			[]string{"reason"},
		),
		StreamingConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "gateway_streaming_connections",
				Help: "Current number of active streaming connections",
			},
		),
		BackendHealth: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gateway_backend_health",
				Help: "Backend health status (1=healthy, 0=unhealthy)",
			},
			[]string{"backend_id", "status"},
		),
		BackendRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_backend_requests_total",
				Help: "Total requests routed to each backend by status",
			},
			[]string{"backend_id", "status"},
		),
		BackendLatencySeconds: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gateway_backend_latency_seconds",
				Help:    "Request latency for each backend",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"backend_id"},
		),
		BackendInFlight: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gateway_backend_in_flight",
				Help: "Current in-flight requests for each backend",
			},
			[]string{"backend_id"},
		),
		BackendCircuitOpen: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gateway_backend_circuit_open",
				Help: "Circuit breaker status for each backend (1=open, 0=closed)",
			},
			[]string{"backend_id"},
		),
	}
}
