package resource

import (
	"context"
	"fmt"
	"sync/atomic"

	"sync"
)

// Shareable is a Manager for a shareable resource, it maintains allocation of a single resource and allows multiple callers to acquire it.
type Shareable[T Resource] struct {
	handle *Handle[T]
	lock   sync.Mutex

	create  CreateFunc[T]
	destroy DestroyFunc[T]

	closed atomic.Bool
}

func NewShareable[T Resource](create CreateFunc[T], destroy DestroyFunc[T]) *Shareable[T] {
	return &Shareable[T]{
		create:  create,
		destroy: destroy,
	}
}

// Aqcuire will allocate the shareable resource or return it if it is already allocated
func (s *Shareable[T]) Acquire(ctx context.Context) (*Handle[T], error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.closed.Load() {
		return nil, ErrClosed
	}
	if s.handle != nil {
		h := s.handle
		return h, nil
	}
	r, err := s.create(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate new resource: %w", err)
	}
	s.handle = &Handle[T]{
		resource: r,
		m:        s,
	}
	return s.handle, nil
}

// Release is for Shareable resources is a no-op
func (s *Shareable[T]) Release(_ *Handle[T]) {}

// Destroy will deallocate the resource, the next call to Acquire will allocate a new resource
func (s *Shareable[T]) Destroy(h *Handle[T]) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.handle == nil || h != s.handle {
		// destroy has already been called for the given handle, this is fairly likely with a shared resource
		return nil
	}
	err := s.destroy(h.resource)
	s.handle = nil
	return err
}

// Close blocks future calls to Acquire and destroys the managed resource
func (s *Shareable[T]) Close() error {
	s.closed.Store(true)
	return s.Destroy(s.handle)
}
