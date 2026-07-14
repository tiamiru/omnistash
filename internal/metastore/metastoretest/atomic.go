package metastoretest

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/metastore"
)

var errAtomicTest = errors.New("metadatatest: deliberate rollback")

func ExerciseAtomicContract(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()

	t.Run("Atomic", func(t *testing.T) {
		t.Parallel()

		t.Run("happy path: commits writes when fn returns nil", func(t *testing.T) {
			t.Parallel()
			store := newStore(t)
			mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
				_, err := tx.CreateNamespace(ctx, DefaultName)

				return err
			})

			exists, err := store.NamespaceExists(t.Context(), DefaultName)
			require.NoError(t, err)
			assert.True(t, exists)
		})

		t.Run("error path: rolls back all writes when fn returns an error", func(t *testing.T) {
			t.Parallel()
			store := newStore(t)

			err := store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error {
				_, createErr := tx.CreateNamespace(ctx, DefaultName)
				if createErr != nil {
					return createErr
				}

				return errAtomicTest
			})

			require.ErrorIs(t, err, errAtomicTest)
			exists, checkErr := store.NamespaceExists(t.Context(), DefaultName)
			require.NoError(t, checkErr)
			assert.False(t, exists, "partial write must not persist after rollback")
		})

		t.Run("edge case: rolls back all writes when fn panics", func(t *testing.T) {
			t.Parallel()
			exerciseAtomicPanic(t, newStore)
		})

		t.Run("concurrency: exactly one of N concurrent transactions creates the namespace", func(t *testing.T) {
			t.Parallel()
			exerciseAtomicSerialization(t, newStore)
		})
	})
}

func exerciseAtomicSerialization(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()
	store := newStore(t)

	const workers = 5
	errs := make([]error, workers)
	created := make([]bool, workers)

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := range workers {
		go func(i int) {
			defer wg.Done()
			errs[i] = store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error {
				wasCreated, err := tx.CreateNamespace(ctx, DefaultName)
				if err != nil {
					return err
				}
				created[i] = wasCreated

				return nil
			})
		}(i)
	}

	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "worker %d", i)
	}

	n := 0
	for _, c := range created {
		if c {
			n++
		}
	}
	assert.Equal(t, 1, n, "exactly one transaction should have created the namespace")
}

func exerciseAtomicPanic(t *testing.T, newStore MetadataStoreSetupFunc) {
	t.Helper()
	store := newStore(t)

	func() {
		defer func() {
			recovered := recover()
			assert.NotNil(t, recovered)
		}()

		store.Atomic(t.Context(), func(ctx context.Context, tx metastore.TxOps) error { //nolint:errcheck,gosec
			_, createErr := tx.CreateNamespace(ctx, DefaultName)
			if createErr != nil {
				return createErr
			}

			panic("metadatatest: deliberate panic")
		})
	}()

	exists, err := store.NamespaceExists(t.Context(), DefaultName)
	require.NoError(t, err)
	assert.False(t, exists, "panic mid-transaction must not persist a partial write")

	mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
		_, createErr := tx.CreateNamespace(ctx, OtherName)

		return createErr
	})
	otherExists, err := store.NamespaceExists(t.Context(), OtherName)
	require.NoError(t, err)
	assert.True(t, otherExists)
}
