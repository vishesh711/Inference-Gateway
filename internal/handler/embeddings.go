package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/vishesh/inference-gateway/internal/cache"
	"github.com/vishesh/inference-gateway/internal/engine"
	"github.com/vishesh/inference-gateway/internal/scheduler"
)

// SetBatcher configures the embeddings batcher
func (h *Handler) SetBatcher(batcher *scheduler.EmbeddingBatcher) {
	h.batcher = batcher
}

// HandleEmbeddings processes /v1/embeddings requests with batching
func (h *Handler) HandleEmbeddings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req engine.EmbeddingRequest
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

	// If no batcher, fall back to direct call
	if h.batcher == nil {
		h.handleEmbeddingDirect(w, ctx, &req, cacheKey)
		return
	}

	// Submit to batcher
	respChan := make(chan scheduler.EmbeddingResponse, 1)
	batchReq := &scheduler.EmbeddingRequest{
		Ctx:      ctx,
		Inputs:   req.Input,
		Model:    req.Model,
		Response: respChan,
	}

	if err := h.batcher.Submit(batchReq); err != nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		h.metrics.RejectedTotal.WithLabelValues("batcher_full").Inc()
		return
	}

	// Wait for batched response
	select {
	case resp := <-respChan:
		if resp.Err != nil {
			h.metrics.RequestsTotal.WithLabelValues(req.Model, "error").Inc()
			http.Error(w, resp.Err.Error(), http.StatusInternalServerError)
			return
		}

		// Build response in OpenAI format
		embeddingResp := engine.EmbeddingResponse{
			Object: "list",
			Model:  req.Model,
		}
		for i, emb := range resp.Embeddings {
			embeddingResp.Data = append(embeddingResp.Data, struct {
				Object    string    `json:"object"`
				Embedding []float64 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				Object:    "embedding",
				Embedding: emb,
				Index:     i,
			})
		}
		embeddingResp.Usage.PromptTokens = resp.Tokens
		embeddingResp.Usage.TotalTokens = resp.Tokens

		// Record metrics
		h.metrics.TokensTotal.WithLabelValues("prompt").Add(float64(resp.Tokens))
		h.metrics.RequestsTotal.WithLabelValues(req.Model, "success").Inc()

		// Record cost
		h.accountant.RecordTokens(resp.Tokens, 0)

		// Cache response
		h.cache.Set(cacheKey, embeddingResp)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(embeddingResp)

	case <-ctx.Done():
		http.Error(w, "Request timeout", http.StatusGatewayTimeout)
		h.metrics.RequestsTotal.WithLabelValues(req.Model, "timeout").Inc()
	}
}

// handleEmbeddingDirect processes embedding without batching
func (h *Handler) handleEmbeddingDirect(w http.ResponseWriter, ctx context.Context, req *engine.EmbeddingRequest, cacheKey string) {
	// Acquire scheduler slot
	if err := h.scheduler.Acquire(ctx); err != nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		h.metrics.RejectedTotal.WithLabelValues("scheduler_full").Inc()
		return
	}
	defer h.scheduler.Release()

	h.metrics.InFlight.Set(float64(h.scheduler.InFlight()))
	defer h.metrics.InFlight.Set(float64(h.scheduler.InFlight()))

	// Call engine
	start := time.Now()
	resp, err := h.engine.CreateEmbedding(ctx, req)
	duration := time.Since(start).Seconds()

	if err != nil {
		h.metrics.RequestsTotal.WithLabelValues(req.Model, "error").Inc()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Record metrics
	h.metrics.GenerationSeconds.Observe(duration)
	h.metrics.TokensTotal.WithLabelValues("prompt").Add(float64(resp.Usage.PromptTokens))
	h.metrics.RequestsTotal.WithLabelValues(req.Model, "success").Inc()

	// Record cost
	h.accountant.RecordTokens(resp.Usage.PromptTokens, 0)

	// Cache response
	h.cache.Set(cacheKey, resp)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
