package mock

import (
	"context"

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
