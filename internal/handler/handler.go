package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/vishesh/inference-gateway/internal/cache"
	"github.com/vishesh/inference-gateway/internal/engine"
	"github.com/vishesh/inference-gateway/internal/metrics"
	"github.com/vishesh/inference-gateway/internal/scheduler"
)

// Handler manages HTTP endpoints and request processing
type Handler struct {
	engine     *engine.Client
	admission  *scheduler.AdmissionController
	scheduler  *scheduler.Scheduler
	batcher    *scheduler.EmbeddingBatcher
	cache      *cache.Cache
	metrics    *metrics.Metrics
	accountant interface {
		RecordTokens(promptTokens, completionTokens int)
	}
	reqTimeout time.Duration
}

// New creates a new handler
func New(
	engineClient *engine.Client,
	admission *scheduler.AdmissionController,
	sched *scheduler.Scheduler,
	c *cache.Cache,
	m *metrics.Metrics,
	accountant interface{ RecordTokens(promptTokens, completionTokens int) },
	reqTimeout time.Duration,
) *Handler {
	return &Handler{
		engine:     engineClient,
		admission:  admission,
		scheduler:  sched,
		cache:      c,
		metrics:    m,
		accountant: accountant,
		reqTimeout: reqTimeout,
	}
}

// HandleCompletions processes /v1/completions requests
func (h *Handler) HandleCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req engine.CompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		h.metrics.RejectedTotal.WithLabelValues("invalid_request").Inc()
		return
	}

	// Check cache
	cacheKey := cache.HashRequest(req)
	if cached, ok := h.cache.Get(cacheKey); ok {
		h.metrics.CacheHitsTotal.Inc()
		h.metrics.RequestsTotal.WithLabelValues(req.Model, "cache_hit").Inc()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached)
		return
	}
	h.metrics.CacheMissesTotal.Inc()

	// Create context with deadline
	ctx, cancel := context.WithTimeout(r.Context(), h.reqTimeout)
	defer cancel()

	// Attempt admission
	queueStart := time.Now()
	admitReq, err := h.admission.Admit(ctx)
	if err != nil {
		if err == scheduler.ErrQueueFull {
			w.Header().Set("Retry-After", "5")
			http.Error(w, "Queue full", http.StatusTooManyRequests)
			h.metrics.RejectedTotal.WithLabelValues("queue_full").Inc()
			return
		}
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		h.metrics.RejectedTotal.WithLabelValues("shutdown").Inc()
		return
	}

	// Update queue depth metric
	h.metrics.QueueDepth.Set(float64(h.admission.QueueDepth()))

	// Process in background
	go func() {
		defer func() {
			h.metrics.QueueDepth.Set(float64(h.admission.QueueDepth()))
		}()

		// Wait in queue
		queueWait := time.Since(queueStart).Seconds()
		h.metrics.QueueWaitSeconds.Observe(queueWait)

		// Acquire scheduler slot
		if err := h.scheduler.Acquire(admitReq.Ctx); err != nil {
			admitReq.Response <- scheduler.Response{Err: err}
			return
		}
		defer h.scheduler.Release()

		h.metrics.InFlight.Set(float64(h.scheduler.InFlight()))
		defer h.metrics.InFlight.Set(float64(h.scheduler.InFlight()))

		// Call engine
		genStart := time.Now()
		resp, err := h.engine.CreateCompletion(admitReq.Ctx, &req)
		genDuration := time.Since(genStart).Seconds()

		if err != nil {
			h.metrics.RequestsTotal.WithLabelValues(req.Model, "error").Inc()
			admitReq.Response <- scheduler.Response{Err: err}
			return
		}

		// Record metrics
		h.metrics.GenerationSeconds.Observe(genDuration)
		h.metrics.TimeToFirstTokenSeconds.Observe(genDuration) // Simplified for non-streaming
		h.metrics.TokensTotal.WithLabelValues("prompt").Add(float64(resp.Usage.PromptTokens))
		h.metrics.TokensTotal.WithLabelValues("completion").Add(float64(resp.Usage.CompletionTokens))
		h.metrics.RequestsTotal.WithLabelValues(req.Model, "success").Inc()

		// Record cost
		h.accountant.RecordTokens(resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

		// Cache response
		h.cache.Set(cacheKey, resp)

		admitReq.Response <- scheduler.Response{Result: resp}
	}()

	// Wait for response
	select {
	case result := <-admitReq.Response:
		if result.Err != nil {
			http.Error(w, result.Err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result.Result)
	case <-ctx.Done():
		http.Error(w, "Request timeout", http.StatusGatewayTimeout)
		h.metrics.RequestsTotal.WithLabelValues(req.Model, "timeout").Inc()
	}
}

// HandleChatCompletions processes /v1/chat/completions requests
func (h *Handler) HandleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req engine.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		h.metrics.RejectedTotal.WithLabelValues("invalid_request").Inc()
		return
	}

	// Check cache
	cacheKey := cache.HashRequest(req)
	if cached, ok := h.cache.Get(cacheKey); ok {
		h.metrics.CacheHitsTotal.Inc()
		h.metrics.RequestsTotal.WithLabelValues(req.Model, "cache_hit").Inc()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cached)
		return
	}
	h.metrics.CacheMissesTotal.Inc()

	// Create context with deadline
	ctx, cancel := context.WithTimeout(r.Context(), h.reqTimeout)
	defer cancel()

	// Attempt admission
	queueStart := time.Now()
	admitReq, err := h.admission.Admit(ctx)
	if err != nil {
		if err == scheduler.ErrQueueFull {
			w.Header().Set("Retry-After", "5")
			http.Error(w, "Queue full", http.StatusTooManyRequests)
			h.metrics.RejectedTotal.WithLabelValues("queue_full").Inc()
			return
		}
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		h.metrics.RejectedTotal.WithLabelValues("shutdown").Inc()
		return
	}

	// Update queue depth metric
	h.metrics.QueueDepth.Set(float64(h.admission.QueueDepth()))

	// Process in background
	go func() {
		defer func() {
			h.metrics.QueueDepth.Set(float64(h.admission.QueueDepth()))
		}()

		// Wait in queue
		queueWait := time.Since(queueStart).Seconds()
		h.metrics.QueueWaitSeconds.Observe(queueWait)

		// Acquire scheduler slot
		if err := h.scheduler.Acquire(admitReq.Ctx); err != nil {
			admitReq.Response <- scheduler.Response{Err: err}
			return
		}
		defer h.scheduler.Release()

		h.metrics.InFlight.Set(float64(h.scheduler.InFlight()))
		defer h.metrics.InFlight.Set(float64(h.scheduler.InFlight()))

		// Call engine
		genStart := time.Now()
		resp, err := h.engine.CreateChatCompletion(admitReq.Ctx, &req)
		genDuration := time.Since(genStart).Seconds()

		if err != nil {
			h.metrics.RequestsTotal.WithLabelValues(req.Model, "error").Inc()
			admitReq.Response <- scheduler.Response{Err: err}
			return
		}

		// Record metrics
		h.metrics.GenerationSeconds.Observe(genDuration)
		h.metrics.TimeToFirstTokenSeconds.Observe(genDuration)
		h.metrics.TokensTotal.WithLabelValues("prompt").Add(float64(resp.Usage.PromptTokens))
		h.metrics.TokensTotal.WithLabelValues("completion").Add(float64(resp.Usage.CompletionTokens))
		h.metrics.RequestsTotal.WithLabelValues(req.Model, "success").Inc()

		// Record cost
		h.accountant.RecordTokens(resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

		// Cache response
		h.cache.Set(cacheKey, resp)

		admitReq.Response <- scheduler.Response{Result: resp}
	}()

	// Wait for response
	select {
	case result := <-admitReq.Response:
		if result.Err != nil {
			http.Error(w, result.Err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result.Result)
	case <-ctx.Done():
		http.Error(w, "Request timeout", http.StatusGatewayTimeout)
		h.metrics.RequestsTotal.WithLabelValues(req.Model, "timeout").Inc()
	}
}
