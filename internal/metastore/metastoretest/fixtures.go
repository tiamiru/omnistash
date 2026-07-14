package metastoretest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/metastore"
)

const (
	DefaultName = "library/test"
	OtherName   = "library/other"
)

type MetadataStoreSetupFunc func(t *testing.T) metastore.MetadataStore

func mustAtomic(t *testing.T, store metastore.MetadataStore, fn func(ctx context.Context, tx metastore.TxOps) error) {
	t.Helper()
	err := store.Atomic(t.Context(), fn)
	require.NoError(t, err)
}
