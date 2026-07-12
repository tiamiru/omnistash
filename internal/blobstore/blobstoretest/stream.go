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

	t.Run("SplitUpload", func(t *testing.T) {
		t.Parallel()
		exerciseSplitUpload(t, newStore)
	})

	t.Run("CancelUpload", func(t *testing.T) {
		t.Parallel()
		exerciseCancelUpload(t, newStore)
	})
}

func exerciseInitiateBlobUpload(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("happy path: returns a unique upload ID", func(t *testing.T) {
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

const wrongOffset = 5

func exerciseAppendBlobChunk(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("error path: unknown session returns ErrBlobUploadUnknown", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		_, err := s.AppendBlobChunk(FakeUploadID, 0, strings.NewReader(TestContent))
		require.ErrorIs(t, err, blobstore.ErrBlobUploadUnknown)
	})

	t.Run("error path: negative offset returns ErrBlobUploadInvalid and preserves offset", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		uploadID, err := s.InitiateBlobUpload()
		require.NoError(t, err)

		_, err = s.AppendBlobChunk(uploadID, -1, strings.NewReader(TestContent))
		require.ErrorIs(t, err, blobstore.ErrBlobUploadInvalid)

		requireOffset(t, s, uploadID, 0)
	})

	t.Run("error path: wrong offset returns ErrRangeNotSatisfiable and preserves offset", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		uploadID, err := s.InitiateBlobUpload()
		require.NoError(t, err)

		_, err = s.AppendBlobChunk(uploadID, wrongOffset, strings.NewReader(TestContent))
		require.ErrorIs(t, err, blobstore.ErrRangeNotSatisfiable)

		requireOffset(t, s, uploadID, 0)
	})

	t.Run("happy path: returns new total size and advances offset", func(t *testing.T) {
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
		assert.Equal(t, int64(len(TestContent)), newSize)

		requireOffset(t, s, uploadID, int64(len(TestContent)))
	})
}
func exerciseFinalizeBlobUploadErrors(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("SessionErrors", func(t *testing.T) {
		t.Parallel()
		exerciseFinalizeBlobUploadSessionErrors(t, newStore)
	})

	t.Run("OffsetErrors", func(t *testing.T) {
		t.Parallel()
		exerciseFinalizeBlobUploadOffsetErrors(t, newStore)
	})

	t.Run("SizeErrors", func(t *testing.T) {
		t.Parallel()
		exerciseFinalizeBlobUploadSizeErrors(t, newStore)
	})

	t.Run("ContentErrors", func(t *testing.T) {
		t.Parallel()
		exerciseFinalizeBlobUploadContentErrors(t, newStore)
	})
}

func exerciseFinalizeBlobUploadSessionErrors(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("error path: unknown session returns ErrBlobUploadUnknown", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		err := s.FinalizeBlobUpload(
			FakeUploadID,
			TestDigest,
			int64(len(TestContent)),
			strings.NewReader(TestContent),
			0,
		)
		require.ErrorIs(t, err, blobstore.ErrBlobUploadUnknown)
	})

	t.Run("error path: malformed digest returns ErrInvalidDigest and preserves session", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		uploadID, err := s.InitiateBlobUpload()
		require.NoError(t, err)

		err = s.FinalizeBlobUpload(
			uploadID,
			MalformedDigest,
			int64(len(TestContent)),
			strings.NewReader(TestContent),
			0,
		)
		require.ErrorIs(t, err, blobstore.ErrInvalidDigest)

		requireOffset(t, s, uploadID, 0)
	})
}

func exerciseFinalizeBlobUploadOffsetErrors(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run(
		"error path: wrong final chunk offset on fresh session returns ErrRangeNotSatisfiable and preserves session",
		func(t *testing.T) {
			t.Parallel()
			s := newStore(t, t.Name(), DefaultPartition)

			uploadID, err := s.InitiateBlobUpload()
			require.NoError(t, err)

			// Session is at offset 0; claiming offset 1 must be rejected.
			err = s.FinalizeBlobUpload(uploadID, TestDigest, int64(len(TestContent)), strings.NewReader(TestContent), 1)
			require.ErrorIs(t, err, blobstore.ErrRangeNotSatisfiable)

			requireOffset(t, s, uploadID, 0)
		},
	)

	t.Run(
		"error path: wrong final chunk offset with prior appends returns ErrRangeNotSatisfiable and preserves session",
		func(t *testing.T) {
			t.Parallel()
			s := newStore(t, t.Name(), DefaultPartition)

			uploadID, err := s.InitiateBlobUpload()
			require.NoError(t, err)

			_, err = s.AppendBlobChunk(uploadID, 0, strings.NewReader("{"))
			require.NoError(t, err)

			err = s.FinalizeBlobUpload(uploadID, TestDigest, int64(len(TestContent)), strings.NewReader("}"), 0)
			require.ErrorIs(t, err, blobstore.ErrRangeNotSatisfiable)

			requireOffset(t, s, uploadID, 1)
		},
	)
}

func exerciseFinalizeBlobUploadSizeErrors(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("error path: size too large returns ErrSizeInvalid and leaves session for vacuum", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		uploadID, err := s.InitiateBlobUpload()
		require.NoError(t, err)

		_, err = s.AppendBlobChunk(uploadID, 0, strings.NewReader(TestContent))
		require.NoError(t, err)

		err = s.FinalizeBlobUpload(uploadID, TestDigest, int64(len(TestContent))+1, nil, -1)
		require.ErrorIs(t, err, blobstore.ErrSizeInvalid)

		_, offsetErr := s.GetBlobUploadOffset(uploadID)
		require.NoError(t, offsetErr)

		vacuumStaging(t, s, 0)
		_, offsetErr = s.GetBlobUploadOffset(uploadID)
		require.ErrorIs(t, offsetErr, blobstore.ErrBlobUploadUnknown)
	})

	t.Run("error path: size too small returns ErrSizeInvalid and leaves session for vacuum", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		uploadID, err := s.InitiateBlobUpload()
		require.NoError(t, err)

		_, err = s.AppendBlobChunk(uploadID, 0, strings.NewReader(TestContent))
		require.NoError(t, err)

		err = s.FinalizeBlobUpload(uploadID, TestDigest, int64(len(TestContent))-1, nil, -1)
		require.ErrorIs(t, err, blobstore.ErrSizeInvalid)

		_, offsetErr := s.GetBlobUploadOffset(uploadID)
		require.NoError(t, offsetErr)

		vacuumStaging(t, s, 0)
		_, offsetErr = s.GetBlobUploadOffset(uploadID)
		require.ErrorIs(t, offsetErr, blobstore.ErrBlobUploadUnknown)
	})
}

func exerciseFinalizeBlobUploadContentErrors(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("error path: digest mismatch returns ErrDigestMismatch and leaves session for vacuum", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		uploadID, err := s.InitiateBlobUpload()
		require.NoError(t, err)

		_, err = s.AppendBlobChunk(uploadID, 0, strings.NewReader(TestContent))
		require.NoError(t, err)

		err = s.FinalizeBlobUpload(uploadID, ZeroDigest, int64(len(TestContent)), nil, -1)
		require.ErrorIs(t, err, blobstore.ErrDigestMismatch)

		_, offsetErr := s.GetBlobUploadOffset(uploadID)
		require.NoError(t, offsetErr)

		vacuumStaging(t, s, 0)
		_, offsetErr = s.GetBlobUploadOffset(uploadID)
		require.ErrorIs(t, offsetErr, blobstore.ErrBlobUploadUnknown)
	})

	t.Run("error path: already committed returns ErrBlobCommitted and removes session", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)
		seedTestBlob(t, s)

		uploadID, err := s.InitiateBlobUpload()
		require.NoError(t, err)

		err = s.FinalizeBlobUpload(uploadID, TestDigest, int64(len(TestContent)), strings.NewReader(TestContent), 0)
		require.ErrorIs(t, err, blobstore.ErrBlobCommitted)

		_, offsetErr := s.GetBlobUploadOffset(uploadID)
		require.ErrorIs(t, offsetErr, blobstore.ErrBlobUploadUnknown)
	})
}

func exerciseMonolithicUpload(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("happy path: content in final chunk is committed and session removed", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		uploadID, err := s.InitiateBlobUpload()
		require.NoError(t, err)

		err = s.FinalizeBlobUpload(uploadID, TestDigest, int64(len(TestContent)), strings.NewReader(TestContent), 0)
		require.NoError(t, err)

		assertSessionRemoved(t, s, uploadID)

		rc, size, err := s.GetBlob(TestDigest)
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, rc.Close()) })
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

		err = s.FinalizeBlobUpload(uploadID, TestDigest, int64(len(TestContent)), nil, -1)
		require.NoError(t, err)

		assertSessionRemoved(t, s, uploadID)

		rc, size, err := s.GetBlob(TestDigest)
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, rc.Close()) })
		assert.Equal(t, int64(len(TestContent)), size)
		data, _ := io.ReadAll(rc)
		assert.Equal(t, TestContent, string(data))
	})
}

func exerciseSplitUpload(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run("happy path: partial appends plus final chunk committed and session removed", func(t *testing.T) {
		t.Parallel()
		s := newStore(t, t.Name(), DefaultPartition)

		uploadID, err := s.InitiateBlobUpload()
		require.NoError(t, err)

		_, err = s.AppendBlobChunk(uploadID, 0, strings.NewReader("{"))
		require.NoError(t, err)

		err = s.FinalizeBlobUpload(uploadID, TestDigest, int64(len(TestContent)), strings.NewReader("}"), 1)
		require.NoError(t, err)

		assertSessionRemoved(t, s, uploadID)

		rc, size, err := s.GetBlob(TestDigest)
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, rc.Close()) })
		assert.Equal(t, int64(len(TestContent)), size)
		data, _ := io.ReadAll(rc)
		assert.Equal(t, TestContent, string(data))
	})
}

func exerciseCancelUpload(t *testing.T, newStore BlobStoreSetupFunc) {
	t.Helper()

	t.Run(
		"edge case: cancel removes session; subsequent ops return ErrBlobUploadUnknown; second cancel is no-op",
		func(t *testing.T) {
			t.Parallel()
			s := newStore(t, t.Name(), DefaultPartition)

			uploadID, err := s.InitiateBlobUpload()
			require.NoError(t, err)

			err = s.CancelBlobUpload(uploadID)
			require.NoError(t, err)

			assertSessionRemoved(t, s, uploadID)

			err = s.CancelBlobUpload(uploadID)
			require.NoError(t, err)
		},
	)
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

	err = s.FinalizeBlobUpload(uploadID, TestDigest, int64(len(TestContent)), nil, -1)
	require.ErrorIs(t, err, blobstore.ErrBlobUploadUnknown)
}
