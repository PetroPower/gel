package resource

import (
	"context"
	"errors"
)

type Resource = any

var ErrClosed = errors.New("manager closed")

type Manager[T Resource] interface {
	// Acquire a managed resource from the manager, may block until the resource is available or the provided context is canceled.
	Acquire(context.Context) (*Handle[T], error)
	// Release signals to the manager that the resource is no longer in use by the caller, but can be used by others
	Release(*Handle[T])
	// Destroy signals to the manager that the resource needs to be cleaned up, usually because it is broken
	Destroy(*Handle[T]) error
	// Close blocks future calls to Acquire and destroys all allocated resources
	Close() error
}

type Handle[T Resource] struct {
	resource T
	m        Manager[T]
}

func (h *Handle[T]) Access() T {
	return h.resource
}

func (h *Handle[T]) Release() {
	h.m.Release(h)
}

func (h *Handle[T]) Destroy() error {
	return h.m.Destroy(h)
}

type CreateFunc[T Resource] func(ctx context.Context) (T, error)
type DestroyFunc[T Resource] func(resource T) error
