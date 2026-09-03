package scheduler

import (
	"context"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/semaphore"
)

// TokenBudget manages token-based admission control
type TokenBudget struct {
	capacity     int64 // Total token capacity
	inUse        int64 // Current tokens in use
	sem          *semaphore.Weighted
	mu           sync.RWMutex
	requestCosts map[string]int64 // Track cost per request ID for release
}

// NewTokenBudget creates a token budget scheduler
func NewTokenBudget(capacity int64) *TokenBudget {
	return &TokenBudget{
		capacity:     capacity,
		sem:          semaphore.NewWeighted(capacity),
		requestCosts: make(map[string]int64),
	}
}

// EstimateTokens estimates token count from text (simple heuristic)
// Real implementation would use actual tokenizer
func EstimateTokens(text string) int64 {
	// Rough heuristic: ~4 characters per token
	// This is approximate - GPT models vary from 3-5 chars/token
	return int64(len(text) / 4)
}

// EstimateCompletionCost estimates total token cost for a completion request
func (tb *TokenBudget) EstimateCompletionCost(prompt string, maxTokens int) int64 {
	promptTokens := EstimateTokens(prompt)
	completionTokens := int64(maxTokens)
	
	if completionTokens == 0 {
		completionTokens = 100 // Default assumption
	}
	
	return promptTokens + completionTokens
}

// EstimateChatCost estimates total token cost for a chat completion request
func (tb *TokenBudget) EstimateChatCost(messages []interface{}, maxTokens int) int64 {
	// Estimate tokens from all messages
	var totalChars int
	
	// This is a simplified version - real implementation would properly
	// parse message content and handle different roles
	for _, msg := range messages {
		if m, ok := msg.(map[string]interface{}); ok {
			if content, ok := m["content"].(string); ok {
				totalChars += len(content)
			}
		}
	}
	
	promptTokens := int64(totalChars / 4)
	completionTokens := int64(maxTokens)
	
	if completionTokens == 0 {
		completionTokens = 100
	}
	
	return promptTokens + completionTokens
}

// Acquire acquires tokens from the budget
func (tb *TokenBudget) Acquire(ctx context.Context, requestID string, cost int64) error {
	// Acquire weighted semaphore
	if err := tb.sem.Acquire(ctx, cost); err != nil {
		return err
	}
	
	// Track the cost for this request
	tb.mu.Lock()
	tb.requestCosts[requestID] = cost
	tb.mu.Unlock()
	
	// Update in-use counter
	atomic.AddInt64(&tb.inUse, cost)
	
	return nil
}

// Release releases tokens back to the budget
func (tb *TokenBudget) Release(requestID string) {
	tb.mu.Lock()
	cost, ok := tb.requestCosts[requestID]
	if !ok {
		tb.mu.Unlock()
		return
	}
	delete(tb.requestCosts, requestID)
	tb.mu.Unlock()
	
	// Release semaphore
	tb.sem.Release(cost)
	
	// Update in-use counter
	atomic.AddInt64(&tb.inUse, -cost)
}

// InUse returns current tokens in use
func (tb *TokenBudget) InUse() int64 {
	return atomic.LoadInt64(&tb.inUse)
}

// Capacity returns total token capacity
func (tb *TokenBudget) Capacity() int64 {
	return tb.capacity
}

// Available returns available token budget
func (tb *TokenBudget) Available() int64 {
	inUse := atomic.LoadInt64(&tb.inUse)
	return tb.capacity - inUse
}

// Utilization returns current utilization percentage (0-100)
func (tb *TokenBudget) Utilization() float64 {
	inUse := float64(atomic.LoadInt64(&tb.inUse))
	capacity := float64(tb.capacity)
	if capacity == 0 {
		return 0
	}
	return (inUse / capacity) * 100
}

// HybridScheduler combines token-based and request-based scheduling
type HybridScheduler struct {
	tokenBudget  *TokenBudget
	reqScheduler *Scheduler // Request-count based scheduler
	useTokens    bool       // Whether to use token-based scheduling
}

// NewHybridScheduler creates a scheduler that can use both strategies
func NewHybridScheduler(tokenCapacity int64, maxRequests int64, useTokens bool) *HybridScheduler {
	return &HybridScheduler{
		tokenBudget:  NewTokenBudget(tokenCapacity),
		reqScheduler: NewScheduler(maxRequests),
		useTokens:    useTokens,
	}
}

// AcquireWithCost acquires admission using token cost
func (hs *HybridScheduler) AcquireWithCost(ctx context.Context, requestID string, tokenCost int64) error {
	if hs.useTokens {
		return hs.tokenBudget.Acquire(ctx, requestID, tokenCost)
	}
	// Fall back to request-based
	return hs.reqScheduler.Acquire(ctx)
}

// ReleaseWithCost releases admission
func (hs *HybridScheduler) ReleaseWithCost(requestID string) {
	if hs.useTokens {
		hs.tokenBudget.Release(requestID)
	} else {
		hs.reqScheduler.Release()
	}
}

// Acquire acquires admission with default cost (for backward compatibility)
func (hs *HybridScheduler) Acquire(ctx context.Context) error {
	if hs.useTokens {
		// Use a default cost for simple acquire
		return hs.tokenBudget.Acquire(ctx, "simple", 100) // 100 token default
	}
	return hs.reqScheduler.Acquire(ctx)
}

// Release releases admission (for backward compatibility)
func (hs *HybridScheduler) Release() {
	if hs.useTokens {
		hs.tokenBudget.Release("simple")
	} else {
		hs.reqScheduler.Release()
	}
}

// InFlight returns current in-flight count
func (hs *HybridScheduler) InFlight() int64 {
	if hs.useTokens {
		// Return token usage as "in-flight" metric
		return hs.tokenBudget.InUse()
	}
	return hs.reqScheduler.InFlight()
}

// GetStats returns current scheduling stats
func (hs *HybridScheduler) GetStats() map[string]interface{} {
	if hs.useTokens {
		return map[string]interface{}{
			"mode":         "token",
			"capacity":     hs.tokenBudget.Capacity(),
			"in_use":       hs.tokenBudget.InUse(),
			"available":    hs.tokenBudget.Available(),
			"utilization":  hs.tokenBudget.Utilization(),
		}
	}
	return map[string]interface{}{
		"mode":      "request",
		"in_flight": hs.reqScheduler.InFlight(),
	}
}
