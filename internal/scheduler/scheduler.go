package scheduler

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"
)

// Scheduler controls in-flight concurrency via a weighted semaphore
type Scheduler struct {
	sem      *semaphore.Weighted
	inFlight int64
	mu       sync.Mutex
}

// NewScheduler creates a scheduler with bounded concurrency
func NewScheduler(maxInFlight int64) *Scheduler {
	return &Scheduler{
		sem: semaphore.NewWeighted(maxInFlight),
	}
}

// Acquire blocks until a slot is available, respecting context cancellation
func (s *Scheduler) Acquire(ctx context.Context) error {
	if err := s.sem.Acquire(ctx, 1); err != nil {
		return err
	}
	s.mu.Lock()
	s.inFlight++
	s.mu.Unlock()
	return nil
}

// Release frees a slot
func (s *Scheduler) Release() {
	s.mu.Lock()
	s.inFlight--
	s.mu.Unlock()
	s.sem.Release(1)
}

// InFlight returns the current number of in-flight requests
func (s *Scheduler) InFlight() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inFlight
}
