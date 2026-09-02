package cost

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Accountant tracks token usage and computes costs
type Accountant struct {
	gpuHourlyRate float64
	
	// Prometheus metrics
	costPerMillionTokens prometheus.Gauge
	totalCostDollars     prometheus.Counter
	
	// Internal tracking
	mu                sync.Mutex
	totalPromptTokens int64
	totalCompTokens   int64
	startTime         int64
}

// New creates a new cost accountant
func New(gpuHourlyRate float64) *Accountant {
	return &Accountant{
		gpuHourlyRate: gpuHourlyRate,
		costPerMillionTokens: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "gateway_cost_per_million_tokens",
				Help: "Estimated cost per million tokens in dollars",
			},
		),
		totalCostDollars: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "gateway_total_cost_dollars",
				Help: "Total estimated cost in dollars since startup",
			},
		),
	}
}

// RecordTokens records token usage and updates cost metrics
func (a *Accountant) RecordTokens(promptTokens, completionTokens int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	a.totalPromptTokens += int64(promptTokens)
	a.totalCompTokens += int64(completionTokens)
	
	// Update metrics
	a.updateCostMetrics()
}

// updateCostMetrics computes and exports cost per million tokens
func (a *Accountant) updateCostMetrics() {
	totalTokens := a.totalPromptTokens + a.totalCompTokens
	if totalTokens == 0 {
		return
	}
	
	// Cost per million tokens = (hourly_rate * hours_of_compute) / tokens * 1M
	// Simplified: we compute based on total runtime and throughput
	// For accurate accounting, we'd track actual GPU time per request
	
	// This is a simplified model: cost per million tokens based on average throughput
	// Real implementation would track cumulative GPU time
	if a.gpuHourlyRate > 0 {
		// Estimate: if we can process X tokens per hour, cost per 1M tokens is:
		// (hourly_rate / tokens_per_hour) * 1_000_000
		// This gets updated as we accumulate more data
		costPerM := (a.gpuHourlyRate / float64(totalTokens)) * 1_000_000.0
		a.costPerMillionTokens.Set(costPerM)
	} else {
		a.costPerMillionTokens.Set(0)
	}
}

// GetStats returns current cost statistics
func (a *Accountant) GetStats() Stats {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	totalTokens := a.totalPromptTokens + a.totalCompTokens
	var costPerM float64
	
	if totalTokens > 0 && a.gpuHourlyRate > 0 {
		costPerM = (a.gpuHourlyRate / float64(totalTokens)) * 1_000_000.0
	}
	
	return Stats{
		TotalPromptTokens:     a.totalPromptTokens,
		TotalCompletionTokens: a.totalCompTokens,
		TotalTokens:           totalTokens,
		CostPerMillionTokens:  costPerM,
		GPUHourlyRate:         a.gpuHourlyRate,
	}
}

// Stats holds cost accounting statistics
type Stats struct {
	TotalPromptTokens     int64
	TotalCompletionTokens int64
	TotalTokens           int64
	CostPerMillionTokens  float64
	GPUHourlyRate         float64
}
