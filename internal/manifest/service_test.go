package manifest_test

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/blobstore"
	"github.com/tiamiru/omnistash/internal/manifest"
	"github.com/tiamiru/omnistash/internal/metastore"
	"github.com/tiamiru/omnistash/internal/ocierror"
	mockmeta "github.com/tiamiru/omnistash/internal/testutil/mock"
)

const (
	testNamespace = "myrepo"
	testTag       = "latest"
	testMediaType = "application/vnd.oci.image.manifest.v1+json"
)

func newTestBody() ([]byte, digest.Digest) {
	body := []byte(`{"schemaVersion":2}`)

	return body, digest.FromBytes(body)
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func newTestSvc(t *testing.T) (*manifest.Service, *mockmeta.MetadataStore, *mockmeta.BlobStore) {
	t.Helper()

	ms := &mockmeta.MetadataStore{Tx: &mockmeta.TxOps{}}
	bs := &mockmeta.BlobStore{}

	t.Cleanup(func() {
		ms.AssertExpectations(t)
		ms.Tx.AssertExpectations(t)
		bs.AssertExpectations(t)
	})

	return manifest.NewService(ms, bs, discardLogger()), ms, bs
}

func TestPutManifest(t *testing.T) {
	t.Parallel()

	defaultBody, defaultDigest := newTestBody()

	subjectDigest := digest.FromString("subject")
	subjectBody := fmt.Appendf(nil,
		`{"schemaVersion":2,"subject":{"digest":%q,"size":1,"mediaType":"application/octet-stream"}}`,
		subjectDigest,
	)
	subjectBodyDigest := digest.FromBytes(subjectBody)

	testCases := []struct {
		name       string
		namespace  string
		reference  string
		mediaType  string
		body       []byte
		setup      func(m *mockmeta.MetadataStore, bs *mockmeta.BlobStore)
		wantErr    error
		wantResult *manifest.PutResult
	}{
		{
			name:      "error path: invalid name",
			namespace: "INVALID",
			reference: defaultDigest.String(),
			mediaType: testMediaType,
			body:      defaultBody,
			wantErr:   ocierror.ErrNameInvalid,
		},
		{
			name:      "error path: namespace not found",
			namespace: testNamespace,
			reference: defaultDigest.String(),
			mediaType: testMediaType,
			body:      defaultBody,
			setup: func(m *mockmeta.MetadataStore, _ *mockmeta.BlobStore) {
				m.On("NamespaceExists", mock.Anything, mock.Anything).Return(false, nil)
			},
			wantErr: ocierror.ErrNameUnknown,
		},
		{
			name:      "error path: invalid reference format",
			namespace: testNamespace,
			reference: "sha256:notvalid",
			mediaType: testMediaType,
			body:      defaultBody,
			wantErr:   ocierror.ErrDigestInvalid,
		},
		{
			name:      "error path: digest mismatch",
			namespace: testNamespace,
			reference: digest.FromString("other").String(),
			mediaType: testMediaType,
			body:      defaultBody,
			wantErr:   ocierror.ErrDigestInvalid,
		},
		{
			name:      "error path: invalid JSON body",
			namespace: testNamespace,
			reference: digest.FromBytes([]byte("not json")).String(),
			mediaType: testMediaType,
			body:      []byte("not json"),
			setup: func(m *mockmeta.MetadataStore, _ *mockmeta.BlobStore) {
				m.On("NamespaceExists", mock.Anything, mock.Anything).Return(true, nil)
			},
			wantErr: ocierror.ErrManifestInvalid,
		},
		{
			name:      "error path: missing media type",
			namespace: testNamespace,
			reference: defaultDigest.String(),
			mediaType: "",
			body:      defaultBody,
			setup: func(m *mockmeta.MetadataStore, _ *mockmeta.BlobStore) {
				m.On("NamespaceExists", mock.Anything, mock.Anything).Return(true, nil)
			},
			wantErr: ocierror.ErrManifestInvalid,
		},
		{
			name:      "error path: tag reference",
			namespace: testNamespace,
			reference: testTag,
			mediaType: testMediaType,
			body:      defaultBody,
			wantErr:   ocierror.ErrUnsupported,
		},
		{
			name:      "happy path: digest reference",
			namespace: testNamespace,
			reference: defaultDigest.String(),
			mediaType: testMediaType,
			body:      defaultBody,
			setup: func(m *mockmeta.MetadataStore, bs *mockmeta.BlobStore) {
				m.On("NamespaceExists", mock.Anything, mock.Anything).Return(true, nil)
				m.On("Atomic", mock.Anything, mock.Anything).Return(nil)
				m.Tx.On("InsertManifest", mock.Anything, testNamespace, defaultDigest, testMediaType, int64(len(defaultBody))).
					Return(nil)
				bs.On("PutBlob", testNamespace, defaultDigest, int64(len(defaultBody)), mock.Anything).
					Return(int64(len(defaultBody)), nil)
			},
			wantResult: &manifest.PutResult{
				Digest:   defaultDigest,
				Location: fmt.Sprintf("/v2/%s/manifests/%s", testNamespace, defaultDigest),
			},
		},
		{
			name:      "happy path: blob already committed",
			namespace: testNamespace,
			reference: defaultDigest.String(),
			mediaType: testMediaType,
			body:      defaultBody,
			setup: func(m *mockmeta.MetadataStore, bs *mockmeta.BlobStore) {
				m.On("NamespaceExists", mock.Anything, mock.Anything).Return(true, nil)
				m.On("Atomic", mock.Anything, mock.Anything).Return(nil)
				m.Tx.On("InsertManifest", mock.Anything, testNamespace, defaultDigest, testMediaType, int64(len(defaultBody))).
					Return(nil)
				bs.On("PutBlob", testNamespace, defaultDigest, int64(len(defaultBody)), mock.Anything).
					Return(int64(0), blobstore.ErrBlobCommitted)
			},
			wantResult: &manifest.PutResult{
				Digest:   defaultDigest,
				Location: fmt.Sprintf("/v2/%s/manifests/%s", testNamespace, defaultDigest),
			},
		},
		{
			name:      "happy path: subject field",
			namespace: testNamespace,
			reference: subjectBodyDigest.String(),
			mediaType: testMediaType,
			body:      subjectBody,
			setup: func(m *mockmeta.MetadataStore, bs *mockmeta.BlobStore) {
				m.On("NamespaceExists", mock.Anything, mock.Anything).Return(true, nil)
				m.On("Atomic", mock.Anything, mock.Anything).Return(nil)
				m.Tx.On("InsertManifest", mock.Anything, testNamespace, subjectBodyDigest, testMediaType, int64(len(subjectBody))).
					Return(nil)
				m.Tx.On("UpsertReferrer", mock.Anything, testNamespace, mock.Anything).Return(nil)
				bs.On("PutBlob", testNamespace, subjectBodyDigest, int64(len(subjectBody)), mock.Anything).
					Return(int64(len(subjectBody)), nil)
			},
			wantResult: &manifest.PutResult{
				Digest:   subjectBodyDigest,
				Location: fmt.Sprintf("/v2/%s/manifests/%s", testNamespace, subjectBodyDigest),
				Subject:  &subjectDigest,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, ms, bs := newTestSvc(t)

			if tc.setup != nil {
				tc.setup(ms, bs)
			}

			res, err := svc.PutManifest(t.Context(), tc.namespace, tc.reference, tc.mediaType, tc.body)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)

			if tc.wantResult != nil {
				assert.Equal(t, *tc.wantResult, res)
			}
		})
	}
}

func TestGetManifest(t *testing.T) {
	t.Parallel()

	defaultBody, defaultDigest := newTestBody()
	manifestDigest := digest.FromString("metastoretest-manifest")

	testCases := []struct {
		name      string
		namespace string
		reference string
		setup     func(m *mockmeta.MetadataStore, bs *mockmeta.BlobStore)
		wantErr   error
		wantInfo  *manifest.ManifestInfo
		wantBody  []byte
	}{
		{
			name:      "error path: invalid name",
			namespace: "INVALID",
			reference: manifestDigest.String(),
			wantErr:   ocierror.ErrNameInvalid,
		},
		{
			name:      "error path: namespace not found",
			namespace: testNamespace,
			reference: manifestDigest.String(),
			setup: func(m *mockmeta.MetadataStore, _ *mockmeta.BlobStore) {
				m.On("NamespaceExists", mock.Anything, mock.Anything).Return(false, nil)
			},
			wantErr: ocierror.ErrNameUnknown,
		},
		{
			name:      "error path: manifest not found by digest",
			namespace: testNamespace,
			reference: manifestDigest.String(),
			setup: func(m *mockmeta.MetadataStore, _ *mockmeta.BlobStore) {
				m.On("NamespaceExists", mock.Anything, mock.Anything).Return(true, nil)
				m.On("Atomic", mock.Anything, mock.Anything).Return(nil)
				m.Tx.On("GetManifestByDigest", mock.Anything, testNamespace, manifestDigest).
					Return(metastore.ManifestRow{}, fmt.Errorf("%w", ocierror.ErrManifestUnknown))
			},
			wantErr: ocierror.ErrManifestUnknown,
		},
		{
			name:      "error path: tag reference",
			namespace: testNamespace,
			reference: testTag,
			wantErr:   ocierror.ErrUnsupported,
		},
		{
			name:      "happy path: by digest",
			namespace: testNamespace,
			reference: defaultDigest.String(),
			setup: func(m *mockmeta.MetadataStore, bs *mockmeta.BlobStore) {
				row := metastore.ManifestRow{
					Namespace: testNamespace,
					Digest:    defaultDigest,
					MediaType: testMediaType,
					Size:      int64(len(defaultBody)),
				}
				m.On("NamespaceExists", mock.Anything, mock.Anything).Return(true, nil)
				m.On("Atomic", mock.Anything, mock.Anything).Return(nil)
				m.Tx.On("GetManifestByDigest", mock.Anything, testNamespace, defaultDigest).Return(row, nil)
				bs.On("GetBlob", testNamespace, defaultDigest).
					Return(io.NopCloser(bytes.NewReader(defaultBody)), int64(len(defaultBody)), nil)
			},
			wantInfo: &manifest.ManifestInfo{
				Digest:    defaultDigest,
				MediaType: testMediaType,
				Size:      int64(len(defaultBody)),
			},
			wantBody: defaultBody,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, ms, bs := newTestSvc(t)

			if tc.setup != nil {
				tc.setup(ms, bs)
			}

			info, rc, err := svc.GetManifest(t.Context(), tc.namespace, tc.reference)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)
			defer rc.Close()

			if tc.wantInfo != nil {
				assert.Equal(t, *tc.wantInfo, info)
				got, _ := io.ReadAll(rc)
				assert.Equal(t, tc.wantBody, got)
			}
		})
	}
}

func TestHeadManifest(t *testing.T) {
	t.Parallel()

	defaultBody, defaultDigest := newTestBody()
	manifestDigest := digest.FromString("metastoretest-manifest")

	testCases := []struct {
		name      string
		namespace string
		reference string
		setup     func(m *mockmeta.MetadataStore, bs *mockmeta.BlobStore)
		wantErr   error
		wantInfo  *manifest.ManifestInfo
	}{
		{
			name:      "error path: invalid name",
			namespace: "INVALID",
			reference: manifestDigest.String(),
			wantErr:   ocierror.ErrNameInvalid,
		},
		{
			name:      "error path: namespace not found",
			namespace: testNamespace,
			reference: manifestDigest.String(),
			setup: func(m *mockmeta.MetadataStore, _ *mockmeta.BlobStore) {
				m.On("NamespaceExists", mock.Anything, mock.Anything).Return(false, nil)
			},
			wantErr: ocierror.ErrNameUnknown,
		},
		{
			name:      "error path: manifest not found by digest",
			namespace: testNamespace,
			reference: manifestDigest.String(),
			setup: func(m *mockmeta.MetadataStore, _ *mockmeta.BlobStore) {
				m.On("NamespaceExists", mock.Anything, mock.Anything).Return(true, nil)
				m.On("Atomic", mock.Anything, mock.Anything).Return(nil)
				m.Tx.On("GetManifestByDigest", mock.Anything, testNamespace, manifestDigest).
					Return(metastore.ManifestRow{}, fmt.Errorf("%w", ocierror.ErrManifestUnknown))
			},
			wantErr: ocierror.ErrManifestUnknown,
		},
		{
			name:      "error path: tag reference",
			namespace: testNamespace,
			reference: testTag,
			wantErr:   ocierror.ErrUnsupported,
		},
		{
			name:      "happy path: by digest",
			namespace: testNamespace,
			reference: defaultDigest.String(),
			setup: func(m *mockmeta.MetadataStore, _ *mockmeta.BlobStore) {
				row := metastore.ManifestRow{
					Namespace: testNamespace,
					Digest:    defaultDigest,
					MediaType: testMediaType,
					Size:      int64(len(defaultBody)),
				}
				m.On("NamespaceExists", mock.Anything, mock.Anything).Return(true, nil)
				m.On("Atomic", mock.Anything, mock.Anything).Return(nil)
				m.Tx.On("GetManifestByDigest", mock.Anything, testNamespace, defaultDigest).Return(row, nil)
			},
			wantInfo: &manifest.ManifestInfo{
				Digest:    defaultDigest,
				MediaType: testMediaType,
				Size:      int64(len(defaultBody)),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, ms, bs := newTestSvc(t)

			if tc.setup != nil {
				tc.setup(ms, bs)
			}

			info, err := svc.HeadManifest(t.Context(), tc.namespace, tc.reference)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)

			if tc.wantInfo != nil {
				assert.Equal(t, *tc.wantInfo, info)
			}
		})
	}
}

func TestDeleteManifest(t *testing.T) {
	t.Parallel()

	_, defaultDigest := newTestBody()

	testCases := []struct {
		name      string
		namespace string
		reference string
		setup     func(m *mockmeta.MetadataStore, bs *mockmeta.BlobStore)
		wantErr   error
	}{
		{
			name:      "error path: invalid name",
			namespace: "INVALID",
			reference: defaultDigest.String(),
			wantErr:   ocierror.ErrNameInvalid,
		},
		{
			name:      "error path: namespace not found",
			namespace: testNamespace,
			reference: defaultDigest.String(),
			setup: func(m *mockmeta.MetadataStore, _ *mockmeta.BlobStore) {
				m.On("NamespaceExists", mock.Anything, mock.Anything).Return(false, nil)
			},
			wantErr: ocierror.ErrNameUnknown,
		},
		{
			name:      "error path: tag reference",
			namespace: testNamespace,
			reference: testTag,
			wantErr:   ocierror.ErrUnsupported,
		},
		{
			name:      "error path: digest not found",
			namespace: testNamespace,
			reference: defaultDigest.String(),
			setup: func(m *mockmeta.MetadataStore, _ *mockmeta.BlobStore) {
				m.On("NamespaceExists", mock.Anything, mock.Anything).Return(true, nil)
				m.On("Atomic", mock.Anything, mock.Anything).Return(nil)
				m.Tx.On("DeleteManifestByDigest", mock.Anything, testNamespace, defaultDigest).
					Return(fmt.Errorf("%w", ocierror.ErrManifestUnknown))
			},
			wantErr: ocierror.ErrManifestUnknown,
		},
		{
			name:      "happy path: by digest",
			namespace: testNamespace,
			reference: defaultDigest.String(),
			setup: func(m *mockmeta.MetadataStore, bs *mockmeta.BlobStore) {
				m.On("NamespaceExists", mock.Anything, mock.Anything).Return(true, nil)
				m.On("Atomic", mock.Anything, mock.Anything).Return(nil)
				m.Tx.On("DeleteManifestByDigest", mock.Anything, testNamespace, defaultDigest).Return(nil)
				m.Tx.On("DeleteReferrer", mock.Anything, testNamespace, defaultDigest).Return(nil)
				bs.On("DeleteBlob", mock.Anything, testNamespace, defaultDigest).Return(nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, ms, bs := newTestSvc(t)

			if tc.setup != nil {
				tc.setup(ms, bs)
			}

			err := svc.DeleteManifest(t.Context(), tc.namespace, tc.reference)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)
		})
	}
}
