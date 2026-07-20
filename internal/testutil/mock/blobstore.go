package mock

import (
	"context"
	"io"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/mock"

	"github.com/tiamiru/omnistash/internal/manifest"
)

var _ manifest.BlobStore = &BlobStore{}

// BlobStore is a testify/mock implementation of manifest.BlobStore.
type BlobStore struct {
	mock.Mock
}

func (m *BlobStore) PutBlob(namespace string, d digest.Digest, size int64, r io.Reader) (int64, error) {
	args := m.Called(namespace, d, size, r)

	n, ok := args.Get(0).(int64)
	if !ok {
		panic("mock: PutBlob: args.Get(0) is not int64")
	}

	return n, args.Error(1)
}

func (m *BlobStore) GetBlob(namespace string, d digest.Digest) (io.ReadCloser, int64, error) {
	args := m.Called(namespace, d)

	rc, _ := args.Get(0).(io.ReadCloser)

	n, ok := args.Get(1).(int64)
	if !ok {
		panic("mock: GetBlob: args.Get(1) is not int64")
	}

	const errIdx = 2

	return rc, n, args.Error(errIdx)
}

func (m *BlobStore) DeleteBlob(ctx context.Context, namespace string, d digest.Digest) error {
	args := m.Called(ctx, namespace, d)

	return args.Error(0)
}
