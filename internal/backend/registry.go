package backend

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BackendStatus represents the health status of a backend
type BackendStatus string

const (
	StatusHealthy   BackendStatus = "healthy"
	StatusDegraded  BackendStatus = "degraded"
	StatusUnhealthy BackendStatus = "unhealthy"
)

// Backend represents a single inference engine backend
type Backend struct {
	ID              string
	URL             string
	Model           string
	Capacity        int           // Max in-flight requests this backend can handle
	Status          BackendStatus
	CurrentLoad     int           // Current in-flight requests
	P95Latency      time.Duration // Recent p95 latency
	FailureCount    int
	SuccessCount    int
	LastHealthCheck time.Time
	CircuitOpen     bool
	mu              sync.RWMutex
}

// GetLoad returns the current load safely
func (b *Backend) GetLoad() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.CurrentLoad
}

// IncrementLoad increases the current load
func (b *Backend) IncrementLoad() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.CurrentLoad++
}

// DecrementLoad decreases the current load
func (b *Backend) DecrementLoad() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.CurrentLoad > 0 {
		b.CurrentLoad--
	}
}

// UpdateLatency updates the p95 latency estimate
func (b *Backend) UpdateLatency(latency time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// Simple exponential moving average
	if b.P95Latency == 0 {
		b.P95Latency = latency
	} else {
		// 80% old, 20% new
		b.P95Latency = time.Duration(float64(b.P95Latency)*0.8 + float64(latency)*0.2)
	}
}

// RecordSuccess records a successful request
func (b *Backend) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.SuccessCount++
	b.FailureCount = 0 // Reset failure count on success
	if b.CircuitOpen && b.SuccessCount >= 3 {
		// Close circuit after 3 consecutive successes
		b.CircuitOpen = false
		b.Status = StatusHealthy
	}
}

// RecordFailure records a failed request
func (b *Backend) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.FailureCount++
	b.SuccessCount = 0 // Reset success count on failure
	
	// Open circuit breaker after 3 consecutive failures
	if b.FailureCount >= 3 {
		b.CircuitOpen = true
		b.Status = StatusUnhealthy
	}
}

// IsHealthy returns true if the backend is healthy and circuit is closed
func (b *Backend) IsHealthy() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Status == StatusHealthy && !b.CircuitOpen
}

// IsCircuitOpen returns true if the circuit breaker is open
func (b *Backend) IsCircuitOpen() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.CircuitOpen
}

// Score calculates a routing score (lower is better)
func (b *Backend) Score() float64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	if !b.IsHealthy() {
		return 1e9 // Very high score for unhealthy backends
	}
	
	// Score = load percentage + latency weight
	loadRatio := float64(b.CurrentLoad) / float64(b.Capacity)
	latencyWeight := b.P95Latency.Seconds() * 10 // Weight latency heavily
	
	return loadRatio + latencyWeight
}

// Registry manages multiple backend instances
type Registry struct {
	backends map[string]*Backend
	mu       sync.RWMutex
}

// NewRegistry creates a new backend registry
func NewRegistry() *Registry {
	return &Registry{
		backends: make(map[string]*Backend),
	}
}

// Register adds a backend to the registry
func (r *Registry) Register(backend *Backend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.backends[backend.ID] = backend
}

// Get retrieves a backend by ID
func (r *Registry) Get(id string) (*Backend, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	backend, ok := r.backends[id]
	return backend, ok
}

// GetAll returns all backends
func (r *Registry) GetAll() []*Backend {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	backends := make([]*Backend, 0, len(r.backends))
	for _, backend := range r.backends {
		backends = append(backends, backend)
	}
	return backends
}

// GetHealthy returns all healthy backends
func (r *Registry) GetHealthy() []*Backend {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	healthy := make([]*Backend, 0)
	for _, backend := range r.backends {
		if backend.IsHealthy() {
			healthy = append(healthy, backend)
		}
	}
	return healthy
}

// SelectBackend selects the best backend using weighted least-loaded strategy
func (r *Registry) SelectBackend() (*Backend, error) {
	healthy := r.GetHealthy()
	
	if len(healthy) == 0 {
		return nil, fmt.Errorf("no healthy backends available")
	}
	
	// Find backend with lowest score
	best := healthy[0]
	bestScore := best.Score()
	
	for _, backend := range healthy[1:] {
		score := backend.Score()
		if score < bestScore {
			best = backend
			bestScore = score
		}
	}
	
	return best, nil
}

// HealthChecker manages health check loop
type HealthChecker struct {
	registry       *Registry
	checkInterval  time.Duration
	checkTimeout   time.Duration
	stopChan       chan struct{}
	pingFunc       func(ctx context.Context, url string) error
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(registry *Registry, checkInterval, checkTimeout time.Duration, pingFunc func(context.Context, string) error) *HealthChecker {
	return &HealthChecker{
		registry:      registry,
		checkInterval: checkInterval,
		checkTimeout:  checkTimeout,
		stopChan:      make(chan struct{}),
		pingFunc:      pingFunc,
	}
}

// Start begins the health check loop
func (hc *HealthChecker) Start() {
	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()
	
	// Run initial health check immediately
	hc.runHealthChecks()
	
	for {
		select {
		case <-ticker.C:
			hc.runHealthChecks()
		case <-hc.stopChan:
			return
		}
	}
}

// Stop stops the health check loop
func (hc *HealthChecker) Stop() {
	close(hc.stopChan)
}

// runHealthChecks checks all backends
func (hc *HealthChecker) runHealthChecks() {
	backends := hc.registry.GetAll()
	
	for _, backend := range backends {
		go hc.checkBackend(backend)
	}
}

// checkBackend performs a health check on a single backend
func (hc *HealthChecker) checkBackend(backend *Backend) {
	ctx, cancel := context.WithTimeout(context.Background(), hc.checkTimeout)
	defer cancel()
	
	err := hc.pingFunc(ctx, backend.URL)
	
	backend.mu.Lock()
	backend.LastHealthCheck = time.Now()
	backend.mu.Unlock()
	
	if err != nil {
		backend.RecordFailure()
	} else {
		backend.RecordSuccess()
	}
}
