package resource

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/PetroPower/gel/smap"
	"golang.org/x/sync/semaphore"
)

// Pool is a Manager that will maintain up to the specified quantity of the managed resource.
type Pool[T Resource] struct {
	items  *smap.Map[*Handle[T], bool]
	sem    *semaphore.Weighted
	closed atomic.Bool

	create  CreateFunc[T]
	destroy DestroyFunc[T]
}

func NewPool[T Resource](create CreateFunc[T], destroy DestroyFunc[T], capacity int) (*Pool[T], error) {
	if capacity < 1 {
		return nil, fmt.Errorf("capacity must be 1 or greater")
	}
	return &Pool[T]{
		items:   smap.Make[*Handle[T], bool](capacity),
		sem:     semaphore.NewWeighted(int64(capacity)),
		create:  create,
		destroy: destroy,
	}, nil
}

// Acquire will wait until capacity is available on the pool or the provided context is canceled.
//
//   - If the context is canceled before capacity is available, ctx.Err() will be returned.
//   - If capacity becomes available and there is already a resource allocated, it will be returned.
//   - If capacity becomes available and the resource needs to be allocated, then the proveded CreateFunc
//     will be called. On success, the created resource will be returned. On failure, the error from
//     the CreateFunc will be returned, and the acquired capacity will be released.
func (p *Pool[T]) Acquire(ctx context.Context) (*Handle[T], error) {
	err := p.sem.Acquire(ctx, 1)
	if err != nil {
		return nil, fmt.Errorf("error while waiting for capacity: %w", err)
	}
	var h *Handle[T]

	err = p.items.Do(func(items map[*Handle[T]]bool) error {
		if p.closed.Load() {
			// pool was closed before we acquired the semaphor or map
			return ErrClosed
		}

		// check for pre-allocated resources
		for existing, available := range items {
			if available {
				// If we found one, mark it as unavailable, and return
				h = existing
				items[existing] = false
				return nil
			}
		}

		// If we didn't find a pre-allocated resource, we need to create a new one
		r, err := p.create(ctx)
		if err != nil {
			return fmt.Errorf("failed to allocate new resource: %w", err)
		}

		h = &Handle[T]{
			resource: r,
			m:        p,
		}
		items[h] = false
		return nil
	})

	if err != nil {
		// put pack semaphore resource, since we failed to acquire a resource
		p.sem.Release(1)
		return nil, err
	}
	return h, nil
}

// Release will place the given resource back into the pool, making it available to other callers of Acquire
func (p *Pool[T]) Release(h *Handle[T]) {
	ok := p.items.Update(h, true)
	if !ok {
		// Resource was destroyed before calling release.
		// The calling code likely deferred Release before encountering an error that warranted destroying the resource.
		return
	}
	p.sem.Release(1)
}

// Destroy will pass the given resource to the provided DestroyFunc and remove it from the pool, freeing up capacity for
// a new resource to be allocated.
func (p *Pool[T]) Destroy(h *Handle[T]) error {
	available, ok := p.items.GetAndDelete(h)
	if !ok {
		// Resource was already destroyed.
		// Pool was likely closed
		return nil
	}
	err := p.destroy(h.resource)
	if !available {
		// if the resource had been acquired, allow a new instance to be created
		p.sem.Release(1)
	}
	return err
}

// Close blocks future calls to Acquire and destroys all allocated resources
func (p *Pool[T]) Close() error {
	alreadyClosed := p.closed.Swap(true)
	if alreadyClosed {
		return nil
	}
	closeErrs := []error{}
	p.items.RangeAndDelete(func(h *Handle[T], available bool) (del bool, cont bool) {
		err := p.destroy(h.resource)
		if !available {
			// Unblock Resource waiters in Acquire
			p.sem.Release(1)
		}
		closeErrs = append(closeErrs, err)
		return true, true
	})
	return errors.Join(closeErrs...)
}
