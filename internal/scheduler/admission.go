package scheduler

import (
	"context"
	"errors"
	"time"
)

var (
	ErrQueueFull = errors.New("queue is full")
	ErrShutdown  = errors.New("scheduler is shutting down")
)

// AdmissionController manages a bounded queue for incoming requests
type AdmissionController struct {
	queue    chan *Request
	shutdown chan struct{}
}

// Request wraps a context and a response channel for async processing
type Request struct {
	Ctx      context.Context
	Response chan Response
}

// Response carries the result or error back to the caller
type Response struct {
	Result interface{}
	Err    error
}

// NewAdmissionController creates a bounded queue
func NewAdmissionController(queueSize int) *AdmissionController {
	return &AdmissionController{
		queue:    make(chan *Request, queueSize),
		shutdown: make(chan struct{}),
	}
}

// Admit tries to enqueue a request, returning ErrQueueFull if at capacity
func (a *AdmissionController) Admit(ctx context.Context) (*Request, error) {
	select {
	case <-a.shutdown:
		return nil, ErrShutdown
	default:
	}

	respChan := make(chan Response, 1)
	req := &Request{
		Ctx:      ctx,
		Response: respChan,
	}

	select {
	case a.queue <- req:
		return req, nil
	default:
		return nil, ErrQueueFull
	}
}

// Next blocks until a request is available or shutdown
func (a *AdmissionController) Next() (*Request, error) {
	select {
	case req := <-a.queue:
		return req, nil
	case <-a.shutdown:
		return nil, ErrShutdown
	}
}

// Shutdown stops accepting new requests and drains the queue
func (a *AdmissionController) Shutdown(timeout time.Duration) {
	close(a.shutdown)
	
	// Drain remaining requests
	deadline := time.After(timeout)
	for {
		select {
		case <-a.queue:
			// Continue draining
		case <-deadline:
			return
		default:
			return
		}
	}
}

// QueueDepth returns current queue size
func (a *AdmissionController) QueueDepth() int {
	return len(a.queue)
}
