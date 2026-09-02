package backend

import (
	"context"
	"fmt"
	"time"

	"github.com/vishesh/inference-gateway/internal/engine"
)

// Router handles routing requests to backends
type Router struct {
	registry *Registry
	clients  map[string]*engine.Client // backend ID -> client
}

// NewRouter creates a new router
func NewRouter(registry *Registry) *Router {
	return &Router{
		registry: registry,
		clients:  make(map[string]*engine.Client),
	}
}

// RegisterClient registers an engine client for a backend
func (r *Router) RegisterClient(backendID string, client *engine.Client) {
	r.clients[backendID] = client
}

// GetClient gets the engine client for a backend
func (r *Router) GetClient(backendID string) (*engine.Client, error) {
	client, ok := r.clients[backendID]
	if !ok {
		return nil, fmt.Errorf("no client for backend %s", backendID)
	}
	return client, nil
}

// RouteCompletion routes a completion request to the best backend
func (r *Router) RouteCompletion(ctx context.Context, req *engine.CompletionRequest) (*engine.CompletionResponse, *Backend, error) {
	backend, err := r.registry.SelectBackend()
	if err != nil {
		return nil, nil, err
	}
	
	client, err := r.GetClient(backend.ID)
	if err != nil {
		return nil, nil, err
	}
	
	// Track load
	backend.IncrementLoad()
	defer backend.DecrementLoad()
	
	// Make request and measure latency
	start := time.Now()
	resp, err := client.CreateCompletion(ctx, req)
	latency := time.Since(start)
	
	// Update backend metrics
	backend.UpdateLatency(latency)
	if err != nil {
		backend.RecordFailure()
		return nil, backend, err
	}
	
	backend.RecordSuccess()
	return resp, backend, nil
}

// RouteChatCompletion routes a chat completion request to the best backend
func (r *Router) RouteChatCompletion(ctx context.Context, req *engine.ChatCompletionRequest) (*engine.ChatCompletionResponse, *Backend, error) {
	backend, err := r.registry.SelectBackend()
	if err != nil {
		return nil, nil, err
	}
	
	client, err := r.GetClient(backend.ID)
	if err != nil {
		return nil, nil, err
	}
	
	// Track load
	backend.IncrementLoad()
	defer backend.DecrementLoad()
	
	// Make request and measure latency
	start := time.Now()
	resp, err := client.CreateChatCompletion(ctx, req)
	latency := time.Since(start)
	
	// Update backend metrics
	backend.UpdateLatency(latency)
	if err != nil {
		backend.RecordFailure()
		return nil, backend, err
	}
	
	backend.RecordSuccess()
	return resp, backend, nil
}

// RouteCompletionStream routes a streaming completion request
func (r *Router) RouteCompletionStream(ctx context.Context, req *engine.CompletionRequest) (<-chan engine.StreamCompletionChunk, <-chan error, *Backend, error) {
	backend, err := r.registry.SelectBackend()
	if err != nil {
		return nil, nil, nil, err
	}
	
	client, err := r.GetClient(backend.ID)
	if err != nil {
		return nil, nil, nil, err
	}
	
	// Track load (will be decremented when stream closes)
	backend.IncrementLoad()
	
	chunkChan, errChan := client.CreateCompletionStream(ctx, req)
	
	// Wrap channels to track metrics
	wrappedChunkChan := make(chan engine.StreamCompletionChunk)
	wrappedErrChan := make(chan error, 1)
	
	go func() {
		defer close(wrappedChunkChan)
		defer close(wrappedErrChan)
		defer backend.DecrementLoad()
		
		start := time.Now()
		var hadError bool
		
		for chunk := range chunkChan {
			wrappedChunkChan <- chunk
		}
		
		// Check for errors
		select {
		case err := <-errChan:
			if err != nil {
				hadError = true
				wrappedErrChan <- err
			}
		default:
		}
		
		// Update backend metrics
		latency := time.Since(start)
		backend.UpdateLatency(latency)
		
		if hadError {
			backend.RecordFailure()
		} else {
			backend.RecordSuccess()
		}
	}()
	
	return wrappedChunkChan, wrappedErrChan, backend, nil
}

// RouteChatCompletionStream routes a streaming chat completion request
func (r *Router) RouteChatCompletionStream(ctx context.Context, req *engine.ChatCompletionRequest) (<-chan engine.StreamChunk, <-chan error, *Backend, error) {
	backend, err := r.registry.SelectBackend()
	if err != nil {
		return nil, nil, nil, err
	}
	
	client, err := r.GetClient(backend.ID)
	if err != nil {
		return nil, nil, nil, err
	}
	
	// Track load (will be decremented when stream closes)
	backend.IncrementLoad()
	
	chunkChan, errChan := client.CreateChatCompletionStream(ctx, req)
	
	// Wrap channels to track metrics
	wrappedChunkChan := make(chan engine.StreamChunk)
	wrappedErrChan := make(chan error, 1)
	
	go func() {
		defer close(wrappedChunkChan)
		defer close(wrappedErrChan)
		defer backend.DecrementLoad()
		
		start := time.Now()
		var hadError bool
		
		for chunk := range chunkChan {
			wrappedChunkChan <- chunk
		}
		
		// Check for errors
		select {
		case err := <-errChan:
			if err != nil {
				hadError = true
				wrappedErrChan <- err
			}
		default:
		}
		
		// Update backend metrics
		latency := time.Since(start)
		backend.UpdateLatency(latency)
		
		if hadError {
			backend.RecordFailure()
		} else {
			backend.RecordSuccess()
		}
	}()
	
	return wrappedChunkChan, wrappedErrChan, backend, nil
}

// RouteEmbedding routes an embedding request
func (r *Router) RouteEmbedding(ctx context.Context, req *engine.EmbeddingRequest) (*engine.EmbeddingResponse, *Backend, error) {
	backend, err := r.registry.SelectBackend()
	if err != nil {
		return nil, nil, err
	}
	
	client, err := r.GetClient(backend.ID)
	if err != nil {
		return nil, nil, err
	}
	
	// Track load
	backend.IncrementLoad()
	defer backend.DecrementLoad()
	
	// Make request and measure latency
	start := time.Now()
	resp, err := client.CreateEmbedding(ctx, req)
	latency := time.Since(start)
	
	// Update backend metrics
	backend.UpdateLatency(latency)
	if err != nil {
		backend.RecordFailure()
		return nil, backend, err
	}
	
	backend.RecordSuccess()
	return resp, backend, nil
}
