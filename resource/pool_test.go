package resource_test

import (
	"context"
	"testing"
	"time"

	"github.com/abferm/gel/resource"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPool(t *testing.T) {
	t.Run("errors if allocated with no capacity", func(t *testing.T) {
		p, err := resource.NewPool(func(ctx context.Context) (int, error) {
			return 0, nil
		},
			func(_ int) error {
				return nil
			},
			0,
		)
		require.Error(t, err)
		require.Nil(t, p)
	})

	manager, err := resource.NewPool[uuid.UUID](func(ctx context.Context) (uuid.UUID, error) {
		return uuid.New(), nil
	},
		func(resource uuid.UUID) error { return nil },
		2,
	)
	require.NoError(t, err)

	t.Run("acquire allocates new resources if none are available", func(t *testing.T) {
		h1, err := manager.Acquire(context.Background())
		require.NoError(t, err)
		defer h1.Release()
		h2, err := manager.Acquire(context.Background())
		require.NoError(t, err)
		defer h2.Release()
		require.NotEqual(t, h1.Access(), h2.Access())
	})

	t.Run("acquire blocks if all resources are checked out", func(t *testing.T) {
		h1, err := manager.Acquire(context.Background())
		require.NoError(t, err)
		defer h1.Release()
		h2, err := manager.Acquire(context.Background())
		require.NoError(t, err)
		defer h2.Release()
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()
		_, err = manager.Acquire(ctx)
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("released resource is the one that is acquired", func(t *testing.T) {
		h1, err := manager.Acquire(context.Background())
		require.NoError(t, err)
		h2, err := manager.Acquire(context.Background())
		require.NoError(t, err)
		defer h2.Release()
		h1Value := h1.Access()
		h1.Release()
		h3, err := manager.Acquire(context.Background())
		require.NoError(t, err)
		defer h3.Release()
		require.Equal(t, h1Value, h3.Access())
	})

	t.Run("destroyed resource is replaced with new", func(t *testing.T) {
		h1, err := manager.Acquire(context.Background())
		require.NoError(t, err)
		h2, err := manager.Acquire(context.Background())
		require.NoError(t, err)
		defer h2.Release()
		h1Value := h1.Access()
		err = h1.Destroy()
		require.NoError(t, err)
		h3, err := manager.Acquire(context.Background())
		require.NoError(t, err)
		defer h3.Release()
		// ensure that h3 does not share value with h1 or h2
		require.NotEqual(t, h1Value, h3.Access())
		require.NotEqual(t, h2.Access(), h3.Access())
	})

	// the following tests operate on a closed Pool manager
	handle, err := manager.Acquire(context.Background())
	require.NoError(t, err)
	defer handle.Release()

	err = manager.Close()
	require.NoError(t, err)
	t.Run("close blocks calls to Acquire", func(t *testing.T) {
		handle, err := manager.Acquire(context.Background())
		require.ErrorIs(t, err, resource.ErrClosed)
		require.Nil(t, handle)
	})

	t.Run("can call Destroy after close", func(t *testing.T) {
		err := handle.Destroy()
		require.NoError(t, err)
	})
}
