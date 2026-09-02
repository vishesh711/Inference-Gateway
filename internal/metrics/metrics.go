package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus collectors
type Metrics struct {
	RequestsTotal           *prometheus.CounterVec
	QueueDepth              prometheus.Gauge
	InFlight                prometheus.Gauge
	QueueWaitSeconds        prometheus.Histogram
	TimeToFirstTokenSeconds prometheus.Histogram
	GenerationSeconds       prometheus.Histogram
	TokensTotal             *prometheus.CounterVec
	CacheHitsTotal          prometheus.Counter
	CacheMissesTotal        prometheus.Counter
	BatchSize               prometheus.Histogram
	RejectedTotal           *prometheus.CounterVec
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
				Help:    "Time from request admission to first token generated",
				Buckets: prometheus.DefBuckets,
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
	}
}
