package metastoretest

import (
	"context"
	"errors"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/metastore"
)

const (
	DefaultName = "library/test"
	OtherName   = "library/other"

	TestSize = int64(2)
)

var (
	TestDigest    = digest.FromString("metastoretest-blob")       //nolint:gochecknoglobals
	OtherDigest   = digest.FromString("metastoretest-other-blob") //nolint:gochecknoglobals
	UnknownDigest = digest.FromString("metastoretest-unknown")    //nolint:gochecknoglobals
)

type MetadataStoreSetupFunc func(t *testing.T) metastore.MetadataStore

func mustAtomic(t *testing.T, store metastore.MetadataStore, fn func(ctx context.Context, tx metastore.TxOps) error) {
	t.Helper()
	err := store.Atomic(t.Context(), fn)
	require.NoError(t, err)
}

func seedNamespace(t *testing.T, store metastore.MetadataStore) {
	t.Helper()
	mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
		_, err := tx.CreateNamespace(ctx, DefaultName)

		return err
	})
}

func seedNamespaceBlob(t *testing.T, store metastore.MetadataStore, d digest.Digest) {
	t.Helper()
	mustAtomic(t, store, func(ctx context.Context, tx metastore.TxOps) error {
		_, err := tx.CreateNamespace(ctx, DefaultName)
		if err != nil && !errors.Is(err, metastore.ErrNameExists) {
			return err
		}

		return tx.InsertNamespaceBlob(ctx, DefaultName, d, TestSize)
	})
}
