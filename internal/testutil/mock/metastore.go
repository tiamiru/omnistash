package mock

import (
	"context"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/mock"

	"github.com/tiamiru/omnistash/internal/metastore"
)

var (
	_ metastore.MetadataStore = &MetadataStore{}
	_ metastore.TxOps         = &TxOps{}
)

// MetadataStore is a testify/mock implementation of metastore.MetadataStore.
type MetadataStore struct {
	mock.Mock

	Tx *TxOps
}

// Atomic records the call for AssertExpectations and delegates to fn(ctx, Tx).
// It returns whatever fn returns so the caller's error-handling logic is exercised.
func (m *MetadataStore) Atomic(ctx context.Context, fn func(ctx context.Context, tx metastore.TxOps) error) error {
	args := m.Called(ctx, fn)

	err := args.Error(0)
	if err != nil {
		return err
	}

	return fn(ctx, m.Tx)
}

func (m *MetadataStore) NamespaceExists(ctx context.Context, name string) (bool, error) {
	args := m.Called(ctx, name)

	return args.Bool(0), args.Error(1)
}

// TxOps is a testify/mock implementation of metastore.TxOps.
type TxOps struct {
	mock.Mock
}

func (m *TxOps) CreateNamespace(ctx context.Context, name string) (metastore.NamespaceRow, error) {
	args := m.Called(ctx, name)

	ns, _ := args.Get(0).(metastore.NamespaceRow)

	return ns, args.Error(1)
}

func (m *TxOps) DeleteNamespace(ctx context.Context, name string) (metastore.NamespaceRow, error) {
	args := m.Called(ctx, name)

	ns, _ := args.Get(0).(metastore.NamespaceRow)

	return ns, args.Error(1)
}

func (m *TxOps) InsertNamespaceBlob(
	ctx context.Context,
	name string,
	d digest.Digest,
	size int64,
) error {
	args := m.Called(ctx, name, d, size)

	return args.Error(0)
}

func (m *TxOps) GetNamespaceBlob(ctx context.Context, name string, d digest.Digest) (int64, error) {
	args := m.Called(ctx, name, d)

	n, ok := args.Get(0).(int64)
	if !ok {
		panic("mock: GetNamespaceBlob: args.Get(0) is not int64")
	}

	return n, args.Error(1)
}

func (m *TxOps) StatNamespaceBlob(
	ctx context.Context,
	name string,
	d digest.Digest,
) (int64, error) {
	args := m.Called(ctx, name, d)

	n, ok := args.Get(0).(int64)
	if !ok {
		panic("mock: StatNamespaceBlob: args.Get(0) is not int64")
	}

	return n, args.Error(1)
}

func (m *TxOps) InsertManifest(
	ctx context.Context,
	namespace string,
	d digest.Digest,
	mediaType string,
	size int64,
) error {
	args := m.Called(ctx, namespace, d, mediaType, size)

	return args.Error(0)
}

func (m *TxOps) GetManifestByDigest(
	ctx context.Context,
	namespace string,
	d digest.Digest,
) (metastore.ManifestRow, error) {
	args := m.Called(ctx, namespace, d)

	row, _ := args.Get(0).(metastore.ManifestRow)

	return row, args.Error(1)
}

func (m *TxOps) DeleteManifestByDigest(ctx context.Context, namespace string, d digest.Digest) error {
	args := m.Called(ctx, namespace, d)

	return args.Error(0)
}
