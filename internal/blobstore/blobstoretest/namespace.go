package blobstoretest

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/blobstore"
)

func ExerciseNamespaceIsolationContract(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("NamespaceIsolation", func(t *testing.T) {
		t.Parallel()
		exerciseNamespaceIsolation(t, newStore)
	})
}

func exerciseNamespaceIsolation(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("edge case: blob written to one namespace is not visible in another", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name())

		seedBlob(t, s, DefaultNamespace, TestDigest, TestContent)

		rc, size, err := s.GetBlob(OtherNamespace, TestDigest)
		require.ErrorIs(t, err, blobstore.ErrBlobUnknown)
		assert.Nil(t, rc)
		assert.Equal(t, int64(0), size)
	})

	t.Run("happy path: same digest is independently committable in two namespaces", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name())

		_, err := s.PutBlob(DefaultNamespace, TestDigest, int64(len(TestContent)), strings.NewReader(TestContent))
		require.NoError(t, err)
		_, err = s.PutBlob(OtherNamespace, TestDigest, int64(len(TestContent)), strings.NewReader(TestContent))
		require.NoError(t, err, "the same digest in a different namespace is a distinct CAS entry")

		sizeDefault, err := s.StatBlob(DefaultNamespace, TestDigest)
		require.NoError(t, err)
		assert.Equal(t, int64(len(TestContent)), sizeDefault)

		_, err = s.StatBlob(OtherNamespace, TestDigest)
		require.NoError(t, err)
	})

	t.Run("edge case: upload session in one namespace is not visible in another", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name())

		uploadID, err := s.InitiateBlobUpload(DefaultNamespace)
		require.NoError(t, err)

		_, err = s.GetBlobUploadOffset(OtherNamespace, uploadID)
		require.ErrorIs(t, err, blobstore.ErrBlobUploadUnknown)
	})
}
