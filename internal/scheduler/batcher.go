package scheduler

import (
	"context"
	"sync"
	"time"
)

// EmbeddingBatch represents a batch of embedding requests
type EmbeddingBatch struct {
	Requests []*EmbeddingRequest
}

// EmbeddingRequest wraps an embedding request with its response channel
type EmbeddingRequest struct {
	Ctx      context.Context
	Inputs   []string
	Model    string
	Response chan<- EmbeddingResponse
}

// EmbeddingResponse carries the embedding result or error
type EmbeddingResponse struct {
	Embeddings [][]float64
	Tokens     int
	Err        error
}

// EmbeddingBatcher coalesces embedding requests into batches
type EmbeddingBatcher struct {
	maxBatchSize int
	maxWaitMs    int
	incoming     chan *EmbeddingRequest
	shutdown     chan struct{}
	wg           sync.WaitGroup
}

// NewEmbeddingBatcher creates a batcher with specified limits
func NewEmbeddingBatcher(maxBatchSize, maxWaitMs int) *EmbeddingBatcher {
	return &EmbeddingBatcher{
		maxBatchSize: maxBatchSize,
		maxWaitMs:    maxWaitMs,
		incoming:     make(chan *EmbeddingRequest, 100),
		shutdown:     make(chan struct{}),
	}
}

// Submit enqueues an embedding request for batching
func (b *EmbeddingBatcher) Submit(req *EmbeddingRequest) error {
	select {
	case b.incoming <- req:
		return nil
	case <-b.shutdown:
		return ErrShutdown
	}
}

// Start begins the batching loop
func (b *EmbeddingBatcher) Start(dispatch func([]*EmbeddingRequest)) {
	b.wg.Add(1)
	go b.run(dispatch)
}

// Shutdown stops the batcher and waits for in-flight batches
func (b *EmbeddingBatcher) Shutdown(timeout time.Duration) {
	close(b.shutdown)
	
	// Wait for the run loop to finish with timeout
	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
	case <-time.After(timeout):
	}
}

// run is the main batching loop
func (b *EmbeddingBatcher) run(dispatch func([]*EmbeddingRequest)) {
	defer b.wg.Done()

	var pending []*EmbeddingRequest
	var timer <-chan time.Time

	for {
		select {
		case req := <-b.incoming:
			pending = append(pending, req)
			
			// Start timer on FIRST request of a batch
			if len(pending) == 1 {
				timer = time.After(time.Duration(b.maxWaitMs) * time.Millisecond)
			}
			
			// Dispatch when batch is full
			if len(pending) >= b.maxBatchSize {
				dispatch(pending)
				pending = nil
				timer = nil
			}

		case <-timer:
			// Timeout: dispatch partial batch
			if len(pending) > 0 {
				dispatch(pending)
				pending = nil
			}
			timer = nil

		case <-b.shutdown:
			// Flush remaining requests on shutdown
			if len(pending) > 0 {
				dispatch(pending)
			}
			return
		}
	}
}
