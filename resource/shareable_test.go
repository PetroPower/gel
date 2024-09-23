package resource_test

import (
	"context"
	"testing"

	"github.com/PetroPower/gel/resource"
	"github.com/stretchr/testify/require"
)

func TestShareable(t *testing.T) {
	i := 0
	manager := resource.NewShareable[int](
		func(ctx context.Context) (int, error) {
			i++
			return i, nil
		},
		func(resource int) error {
			return nil
		},
	)
	handle, err := manager.Acquire(context.Background())
	require.NoError(t, err)
	t.Run("first call allocates new resource", func(t *testing.T) {
		require.Equal(t, 1, handle.Access())
	})

	t.Run("releasing resource does not trigger reallocation", func(t *testing.T) {
		handle.Release()
		handle, err = manager.Acquire(context.Background())
		require.NoError(t, err)
		require.Equal(t, 1, handle.Access())
	})

	t.Run("destroying resource causes next acquire call to reallocate", func(t *testing.T) {
		err := handle.Destroy()
		require.NoError(t, err)
		handle, err = manager.Acquire(context.Background())
		require.NoError(t, err)
		require.Equal(t, 2, handle.Access())
	})

	// the following tests operate on a closed Shareable manager
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
