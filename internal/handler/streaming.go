package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/vishesh/inference-gateway/internal/engine"
	"github.com/vishesh/inference-gateway/internal/scheduler"
)

// HandleCompletionsStreaming handles streaming completion requests
func (h *Handler) HandleCompletionsStreaming(w http.ResponseWriter, r *http.Request, req *engine.CompletionRequest) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Create context with deadline
	ctx, cancel := context.WithTimeout(r.Context(), h.reqTimeout)
	defer cancel()

	// Attempt admission
	queueStart := time.Now()
	admitReq, err := h.admission.Admit(ctx)
	if err != nil {
		if err == scheduler.ErrQueueFull {
			h.writeSSEError(w, flusher, "Queue full", 429)
			h.metrics.RejectedTotal.WithLabelValues("queue_full").Inc()
			return
		}
		h.writeSSEError(w, flusher, "Service unavailable", 503)
		h.metrics.RejectedTotal.WithLabelValues("shutdown").Inc()
		return
	}

	// Wait in queue
	queueWait := time.Since(queueStart).Seconds()
	h.metrics.QueueWaitSeconds.Observe(queueWait)
	h.metrics.QueueDepth.Set(float64(h.admission.QueueDepth()))

	// Acquire scheduler slot
	if err := h.scheduler.Acquire(admitReq.Ctx); err != nil {
		h.writeSSEError(w, flusher, "Scheduler error", 500)
		return
	}
	defer h.scheduler.Release()

	h.metrics.InFlight.Set(float64(h.scheduler.InFlight()))
	defer h.metrics.InFlight.Set(float64(h.scheduler.InFlight()))

	// Start streaming
	genStart := time.Now()
	var firstTokenTime time.Time
	var tokenCount int
	var promptTokens, completionTokens int

	chunkChan, errChan, backend, err := h.router.RouteCompletionStream(admitReq.Ctx, req)
	if err != nil {
		h.writeSSEError(w, flusher, "Routing error: "+err.Error(), 500)
		return
	}

	for {
		select {
		case chunk, ok := <-chunkChan:
			if !ok {
				// Stream finished successfully
				genDuration := time.Since(genStart).Seconds()
				
				// Calculate TPOT (time per output token)
				if tokenCount > 1 && !firstTokenTime.IsZero() {
					tpot := time.Since(firstTokenTime).Seconds() / float64(tokenCount-1)
					h.metrics.TimePerOutputTokenSeconds.Observe(tpot)
				}

				// Record metrics
				h.metrics.GenerationSeconds.Observe(genDuration)
				h.metrics.TokensTotal.WithLabelValues("prompt").Add(float64(promptTokens))
				h.metrics.TokensTotal.WithLabelValues("completion").Add(float64(completionTokens))
				h.metrics.RequestsTotal.WithLabelValues(req.Model, "success").Inc()
				if backend != nil {
					h.metrics.BackendRequestsTotal.WithLabelValues(backend.ID, "success").Inc()
				}
				h.accountant.RecordTokens(promptTokens, completionTokens)

				// Send [DONE]
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()
				return
			}

			// Track first token time (TTFT)
			if tokenCount == 0 {
				firstTokenTime = time.Now()
				ttft := firstTokenTime.Sub(genStart).Seconds()
				h.metrics.TimeToFirstTokenSeconds.Observe(ttft)
			}
			tokenCount++

			// Estimate tokens (rough heuristic)
			if len(chunk.Choices) > 0 {
				completionTokens += len(chunk.Choices[0].Text) / 4 // ~4 chars per token
			}

			// Write chunk
			chunkJSON, err := json.Marshal(chunk)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", chunkJSON)
			flusher.Flush()

		case err := <-errChan:
			if err != nil {
				h.metrics.RequestsTotal.WithLabelValues(req.Model, "error").Inc()
				h.writeSSEError(w, flusher, err.Error(), 500)
				return
			}

		case <-ctx.Done():
			h.metrics.RequestsTotal.WithLabelValues(req.Model, "timeout").Inc()
			h.writeSSEError(w, flusher, "Request timeout", 504)
			return
		}
	}
}

// HandleChatCompletionsStreaming handles streaming chat completion requests
func (h *Handler) HandleChatCompletionsStreaming(w http.ResponseWriter, r *http.Request, req *engine.ChatCompletionRequest) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Create context with deadline
	ctx, cancel := context.WithTimeout(r.Context(), h.reqTimeout)
	defer cancel()

	// Attempt admission
	queueStart := time.Now()
	admitReq, err := h.admission.Admit(ctx)
	if err != nil {
		if err == scheduler.ErrQueueFull {
			h.writeSSEError(w, flusher, "Queue full", 429)
			h.metrics.RejectedTotal.WithLabelValues("queue_full").Inc()
			return
		}
		h.writeSSEError(w, flusher, "Service unavailable", 503)
		h.metrics.RejectedTotal.WithLabelValues("shutdown").Inc()
		return
	}

	// Wait in queue
	queueWait := time.Since(queueStart).Seconds()
	h.metrics.QueueWaitSeconds.Observe(queueWait)
	h.metrics.QueueDepth.Set(float64(h.admission.QueueDepth()))

	// Acquire scheduler slot
	if err := h.scheduler.Acquire(admitReq.Ctx); err != nil {
		h.writeSSEError(w, flusher, "Scheduler error", 500)
		return
	}
	defer h.scheduler.Release()

	h.metrics.InFlight.Set(float64(h.scheduler.InFlight()))
	defer h.metrics.InFlight.Set(float64(h.scheduler.InFlight()))

	// Start streaming
	genStart := time.Now()
	var firstTokenTime time.Time
	var tokenCount int
	var promptTokens, completionTokens int

	chunkChan, errChan, backend, err := h.router.RouteChatCompletionStream(admitReq.Ctx, req)
	if err != nil {
		h.writeSSEError(w, flusher, "Routing error: "+err.Error(), 500)
		return
	}

	for {
		select {
		case chunk, ok := <-chunkChan:
			if !ok {
				// Stream finished successfully
				genDuration := time.Since(genStart).Seconds()
				
				// Calculate TPOT (time per output token)
				if tokenCount > 1 && !firstTokenTime.IsZero() {
					tpot := time.Since(firstTokenTime).Seconds() / float64(tokenCount-1)
					h.metrics.TimePerOutputTokenSeconds.Observe(tpot)
				}

				// Record metrics
				h.metrics.GenerationSeconds.Observe(genDuration)
				h.metrics.TokensTotal.WithLabelValues("prompt").Add(float64(promptTokens))
				h.metrics.TokensTotal.WithLabelValues("completion").Add(float64(completionTokens))
				h.metrics.RequestsTotal.WithLabelValues(req.Model, "success").Inc()
				if backend != nil {
					h.metrics.BackendRequestsTotal.WithLabelValues(backend.ID, "success").Inc()
				}
				h.accountant.RecordTokens(promptTokens, completionTokens)

				// Send [DONE]
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()
				return
			}

			// Track first token time (TTFT)
			if tokenCount == 0 {
				firstTokenTime = time.Now()
				ttft := firstTokenTime.Sub(genStart).Seconds()
				h.metrics.TimeToFirstTokenSeconds.Observe(ttft)
			}
			tokenCount++

			// Estimate tokens (rough heuristic)
			if len(chunk.Choices) > 0 {
				completionTokens += len(chunk.Choices[0].Delta.Content) / 4 // ~4 chars per token
			}

			// Write chunk
			chunkJSON, err := json.Marshal(chunk)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", chunkJSON)
			flusher.Flush()

		case err := <-errChan:
			if err != nil {
				h.metrics.RequestsTotal.WithLabelValues(req.Model, "error").Inc()
				h.writeSSEError(w, flusher, err.Error(), 500)
				return
			}

		case <-ctx.Done():
			h.metrics.RequestsTotal.WithLabelValues(req.Model, "timeout").Inc()
			h.writeSSEError(w, flusher, "Request timeout", 504)
			return
		}
	}
}

// writeSSEError writes an error in SSE format
func (h *Handler) writeSSEError(w http.ResponseWriter, flusher http.Flusher, message string, code int) {
	errorData := map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"type":    "gateway_error",
			"code":    code,
		},
	}
	errorJSON, _ := json.Marshal(errorData)
	fmt.Fprintf(w, "data: %s\n\n", errorJSON)
	flusher.Flush()
}
