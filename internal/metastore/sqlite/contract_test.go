package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/metastore"
	"github.com/tiamiru/omnistash/internal/metastore/metastoretest"
)

func newContractTestStore(t *testing.T) metastore.MetadataStore { //nolint:ireturn
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "meta.db")

	store, err := NewSQLiteMetadataStore(context.Background(), dbPath)
	require.NoError(t, err)

	t.Cleanup(func() {
		closeErr := store.Close()
		require.NoError(t, closeErr)
	})

	err = ApplyMigrations(context.Background(), store)
	require.NoError(t, err)

	err = CheckMigrations(context.Background(), store)
	require.NoError(t, err)

	return store
}

func TestSQLiteMetadataStore_Contract(t *testing.T) {
	t.Parallel()
	metastoretest.ExerciseMetadataStoreContract(t, newContractTestStore)
}
