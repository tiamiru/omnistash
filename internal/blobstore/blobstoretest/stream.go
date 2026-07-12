package blobstoretest

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/blobstore"
)

func ExerciseBlobUploaderContract(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("InitiateBlobUpload", func(t *testing.T) {
		t.Parallel()
		exerciseInitiateBlobUpload(t, newStore)
	})

	t.Run("GetBlobUploadOffset", func(t *testing.T) {
		t.Parallel()
		exerciseGetBlobUploadOffset(t, newStore)
	})

	t.Run("AppendBlobChunk", func(t *testing.T) {
		t.Parallel()
		exerciseAppendBlobChunk(t, newStore)
	})

	t.Run("FinalizeBlobUploadErrors", func(t *testing.T) {
		t.Parallel()
		exerciseFinalizeBlobUploadErrors(t, newStore)
	})

	t.Run("MonolithicUpload", func(t *testing.T) {
		t.Parallel()
		exerciseMonolithicUpload(t, newStore)
	})

	t.Run("ChunkedUpload", func(t *testing.T) {
		t.Parallel()
		exerciseChunkedUpload(t, newStore)
	})

	t.Run("CancelUpload", func(t *testing.T) {
		t.Parallel()
		exerciseCancelUpload(t, newStore)
	})

	t.Run("EmptyBlobUpload", func(t *testing.T) {
		t.Parallel()
		exerciseEmptyBlobUpload(t, newStore)
	})

	t.Run("CrossPartitionIsolation", func(t *testing.T) {
		t.Parallel()
		exerciseCrossPartitionIsolation(t, newStore)
	})
}

func exerciseInitiateBlobUpload(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("happy path: returns unique upload IDs", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		id1, err := s.InitiateBlobUpload()
		require.NoError(t, err)
		assert.NotEmpty(t, id1)

		id2, err := s.InitiateBlobUpload()
		require.NoError(t, err)
		assert.NotEqual(t, id1, id2)
	})
}

func exerciseGetBlobUploadOffset(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("error path: unknown session returns ErrBlobUploadUnknown", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		_, err := s.GetBlobUploadOffset(FakeUploadID)
		require.ErrorIs(t, err, blobstore.ErrBlobUploadUnknown)
	})

	t.Run("happy path: fresh session returns offset 0", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		uploadID, err := s.InitiateBlobUpload()
		require.NoError(t, err)

		requireOffset(t, s, uploadID, 0)
	})
}

func exerciseAppendBlobChunk(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("error path: unknown session returns ErrBlobUploadUnknown", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		_, err := s.AppendBlobChunk(FakeUploadID, 0, strings.NewReader(TestContent))
		require.ErrorIs(t, err, blobstore.ErrBlobUploadUnknown)
	})

	t.Run("error path: wrong offset returns ErrRangeNotSatisfiable", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		uploadID, err := s.InitiateBlobUpload()
		require.NoError(t, err)

		_, err = s.AppendBlobChunk(uploadID, 5, strings.NewReader(TestContent))
		require.ErrorIs(t, err, blobstore.ErrRangeNotSatisfiable)

		requireOffset(t, s, uploadID, 0)
	})

	t.Run("happy path: returns new size and advances offset", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		uploadID, err := s.InitiateBlobUpload()
		require.NoError(t, err)

		newSize, err := s.AppendBlobChunk(uploadID, 0, strings.NewReader("{"))
		require.NoError(t, err)
		assert.Equal(t, int64(1), newSize)

		requireOffset(t, s, uploadID, 1)

		newSize, err = s.AppendBlobChunk(uploadID, 1, strings.NewReader("}"))
		require.NoError(t, err)
		assert.Equal(t, int64(2), newSize)

		requireOffset(t, s, uploadID, int64(len(TestContent)))
	})
}
func exerciseFinalizeBlobUploadErrors(t *testing.T, newStore BlobStoreSetupFunc) { //nolint:funlen
	t.Helper()

	t.Run("error path: unknown session returns ErrBlobUploadUnknown", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		err := s.FinalizeBlobUpload(FakeUploadID, TestDigest, int64(len(TestContent)))
		require.ErrorIs(t, err, blobstore.ErrBlobUploadUnknown)
	})

	t.Run("error path: size mismatch returns ErrSizeInvalid and session survives", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		uploadID, err := s.InitiateBlobUpload()
		require.NoError(t, err)

		_, err = s.AppendBlobChunk(uploadID, 0, strings.NewReader(TestContent))
		require.NoError(t, err)

		err = s.FinalizeBlobUpload(uploadID, TestDigest, int64(len(TestContent))+1)
		require.ErrorIs(t, err, blobstore.ErrSizeInvalid)

		_, offsetErr := s.GetBlobUploadOffset(uploadID)
		require.NoError(t, offsetErr)

		err = s.FinalizeBlobUpload(uploadID, TestDigest, int64(len(TestContent)))
		require.NoError(t, err)

		assertSessionRemoved(t, s, uploadID)
	})

	t.Run("error path: digest mismatch returns ErrDigestMismatch and session remains", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		uploadID, err := s.InitiateBlobUpload()
		require.NoError(t, err)

		_, err = s.AppendBlobChunk(uploadID, 0, strings.NewReader(TestContent))
		require.NoError(t, err)

		err = s.FinalizeBlobUpload(uploadID, ZeroDigest, int64(len(TestContent)))
		require.ErrorIs(t, err, blobstore.ErrDigestMismatch)

		_, offsetErr := s.GetBlobUploadOffset(uploadID)
		require.NoError(t, offsetErr)

		cancelErr := s.CancelBlobUpload(uploadID)
		require.NoError(t, cancelErr)

		assertSessionRemoved(t, s, uploadID)
	})

	t.Run("error path: already committed returns ErrBlobCommitted and removes session", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)
		seedTestBlob(t, s)

		uploadID, err := s.InitiateBlobUpload()
		require.NoError(t, err)

		_, err = s.AppendBlobChunk(uploadID, 0, strings.NewReader(TestContent))
		require.NoError(t, err)

		err = s.FinalizeBlobUpload(uploadID, TestDigest, int64(len(TestContent)))
		require.ErrorIs(t, err, blobstore.ErrBlobCommitted)

		_, offsetErr := s.GetBlobUploadOffset(uploadID)
		require.ErrorIs(t, offsetErr, blobstore.ErrBlobUploadUnknown)
	})
}

func exerciseMonolithicUpload(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("happy path: content committed and session removed", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		uploadID, err := s.InitiateBlobUpload()
		require.NoError(t, err)

		_, err = s.AppendBlobChunk(uploadID, 0, strings.NewReader(TestContent))
		require.NoError(t, err)

		err = s.FinalizeBlobUpload(uploadID, TestDigest, int64(len(TestContent)))
		require.NoError(t, err)

		assertSessionRemoved(t, s, uploadID)

		rc, size, err := s.GetBlob(TestDigest)
		require.NoError(t, err)
		t.Cleanup(func() {
			assert.NoError(t, rc.Close())
		})
		assert.Equal(t, int64(len(TestContent)), size)
		data, _ := io.ReadAll(rc)
		assert.Equal(t, TestContent, string(data))
	})
}

func exerciseChunkedUpload(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("happy path: multiple appends committed and session removed", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		uploadID, err := s.InitiateBlobUpload()
		require.NoError(t, err)

		newSize, err := s.AppendBlobChunk(uploadID, 0, strings.NewReader("{"))
		require.NoError(t, err)
		assert.Equal(t, int64(1), newSize)

		newSize, err = s.AppendBlobChunk(uploadID, 1, strings.NewReader("}"))
		require.NoError(t, err)
		assert.Equal(t, int64(len(TestContent)), newSize)

		err = s.FinalizeBlobUpload(uploadID, TestDigest, int64(len(TestContent)))
		require.NoError(t, err)

		assertSessionRemoved(t, s, uploadID)

		rc, size, err := s.GetBlob(TestDigest)
		require.NoError(t, err)
		t.Cleanup(func() {
			assert.NoError(t, rc.Close())
		})
		assert.Equal(t, int64(len(TestContent)), size)
		data, _ := io.ReadAll(rc)
		assert.Equal(t, TestContent, string(data))
	})
}

func exerciseCancelUpload(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("happy path: cancel removes session and second cancel is a no-op", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		uploadID, err := s.InitiateBlobUpload()
		require.NoError(t, err)

		err = s.CancelBlobUpload(uploadID)
		require.NoError(t, err)

		assertSessionRemoved(t, s, uploadID)

		err = s.CancelBlobUpload(uploadID)
		require.NoError(t, err)
	})
}

func exerciseEmptyBlobUpload(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("happy path: zero-byte blob committed", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		uploadID, err := s.InitiateBlobUpload()
		require.NoError(t, err)

		err = s.FinalizeBlobUpload(uploadID, EmptyDigest, 0)
		require.NoError(t, err)

		assertSessionRemoved(t, s, uploadID)

		rc, size, err := s.GetBlob(EmptyDigest)
		require.NoError(t, err)
		t.Cleanup(func() {
			assert.NoError(t, rc.Close())
		})
		assert.Equal(t, int64(0), size)
		data, _ := io.ReadAll(rc)
		assert.Empty(t, data)
	})
}

func exerciseCrossPartitionIsolation(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("happy path: session in one partition is invisible to another", func(t *testing.T) {
		t.Parallel()
		sA := newStore(t, t.Name(), DefaultPartition)
		sB := newStore(t, t.Name(), OtherPartition)

		uploadID, err := sA.InitiateBlobUpload()
		require.NoError(t, err)

		_, err = sB.GetBlobUploadOffset(uploadID)
		require.ErrorIs(t, err, blobstore.ErrBlobUploadUnknown)
	})
}

func requireOffset(t *testing.T, s blobstore.BlobStore, uploadID string, want int64) {
	t.Helper()

	offset, err := s.GetBlobUploadOffset(uploadID)
	require.NoError(t, err)
	assert.Equal(t, want, offset)
}

func assertSessionRemoved(t *testing.T, s blobstore.BlobStore, uploadID string) {
	t.Helper()

	_, err := s.GetBlobUploadOffset(uploadID)
	require.ErrorIs(t, err, blobstore.ErrBlobUploadUnknown)

	_, err = s.AppendBlobChunk(uploadID, 0, strings.NewReader(TestContent))
	require.ErrorIs(t, err, blobstore.ErrBlobUploadUnknown)

	err = s.FinalizeBlobUpload(uploadID, TestDigest, int64(len(TestContent)))
	require.ErrorIs(t, err, blobstore.ErrBlobUploadUnknown)
}
