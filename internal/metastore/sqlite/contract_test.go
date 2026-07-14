package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/tiamiru/omnistash/internal/metastore"
	"github.com/tiamiru/omnistash/internal/metastore/metastoretest"
)

func newContractTestStore(t *testing.T) metastore.MetadataStore { //nolint:ireturn
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "meta.db")

	store, err := NewSQLiteMetadataStore(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}

	t.Cleanup(func() {
		closeErr := store.Close()
		if closeErr != nil {
			t.Errorf("store.Close: %v", closeErr)
		}
	})

	err = ApplyMigrations(context.Background(), store)
	if err != nil {
		t.Fatalf("ApplyMigrations: %v", err)
	}

	err = CheckMigrations(context.Background(), store)
	if err != nil {
		t.Fatalf("CheckMigrations: %v", err)
	}

	return store
}

func TestSQLiteMetadataStore_Contract(t *testing.T) {
	t.Parallel()
	metastoretest.ExerciseMetadataStoreContract(t, newContractTestStore)
}
