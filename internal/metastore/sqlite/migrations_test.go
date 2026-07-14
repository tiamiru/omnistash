package sqlite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openMemoryStore(t *testing.T) *SQLiteMetadataStore {
	t.Helper()

	store, err := NewSQLiteMetadataStore(context.Background(), ":memory:")
	require.NoError(t, err)

	t.Cleanup(func() {
		closeErr := store.Close()
		if closeErr != nil {
			t.Errorf("store.Close: %v", closeErr)
		}
	})

	return store
}

func TestCheckMigrations(t *testing.T) {
	t.Parallel()

	t.Run("happy path: all tables present after ApplyMigrations", func(t *testing.T) {
		t.Parallel()

		store := openMemoryStore(t)
		err := ApplyMigrations(context.Background(), store)
		require.NoError(t, err)

		err = CheckMigrations(context.Background(), store)
		assert.NoError(t, err)
	})

	t.Run("error path: returns joined errors for every missing table", func(t *testing.T) {
		t.Parallel()

		store := openMemoryStore(t)

		err := CheckMigrations(context.Background(), store)
		require.Error(t, err)

		for _, table := range requiredTables {
			assert.ErrorContains(t, err, table)
		}
	})
}
