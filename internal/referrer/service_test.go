package referrer_test

import (
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/metastore"
	"github.com/tiamiru/omnistash/internal/ocierror"
	"github.com/tiamiru/omnistash/internal/referrer"
	mockmeta "github.com/tiamiru/omnistash/internal/testutil/mock"
)

const (
	testNamespace    = "myrepo"
	testMediaType    = "application/vnd.oci.image.manifest.v1+json"
	testArtifactType = "application/vnd.example.sbom"
)

func newTestSvc(t *testing.T) (*referrer.Service, *mockmeta.MetadataStore) {
	t.Helper()

	ms := &mockmeta.MetadataStore{Tx: &mockmeta.TxOps{}}

	t.Cleanup(func() {
		ms.AssertExpectations(t)
		ms.Tx.AssertExpectations(t)
	})

	return referrer.NewService(ms), ms
}

func TestListReferrers(t *testing.T) {
	t.Parallel()

	subjectDigest := digest.FromString("subject")
	referrerDigest := digest.FromString("referrer")

	fixtureRow := metastore.ReferrerRow{
		SubjectDigest: subjectDigest,
		Digest:        referrerDigest,
		MediaType:     testMediaType,
		ArtifactType:  testArtifactType,
		Size:          42,
		Annotations:   map[string]string{"key": "value"},
	}

	fixtureDescriptor := ocispec.Descriptor{
		MediaType:    testMediaType,
		Digest:       referrerDigest,
		Size:         42,
		ArtifactType: testArtifactType,
		Annotations:  map[string]string{"key": "value"},
	}

	testCases := []struct {
		name         string
		namespace    string
		subject      digest.Digest
		artifactType string
		setup        func(m *mockmeta.MetadataStore)
		wantErr      error
		wantResult   *referrer.ListResult
	}{
		{
			name:      "error path: invalid namespace name",
			namespace: "INVALID",
			subject:   subjectDigest,
			wantErr:   ocierror.ErrNameInvalid,
		},
		{
			name:      "error path: namespace not found",
			namespace: testNamespace,
			subject:   subjectDigest,
			setup: func(m *mockmeta.MetadataStore) {
				m.On("NamespaceExists", mock.Anything, testNamespace).Return(false, nil)
			},
			wantErr: ocierror.ErrNameUnknown,
		},
		{
			name:      "happy path: returns all referrers without filter",
			namespace: testNamespace,
			subject:   subjectDigest,
			setup: func(m *mockmeta.MetadataStore) {
				m.On("NamespaceExists", mock.Anything, testNamespace).Return(true, nil)
				m.On("Atomic", mock.Anything, mock.Anything).Return(nil)
				m.Tx.On("ListReferrers", mock.Anything, testNamespace, subjectDigest).
					Return([]metastore.ReferrerRow{fixtureRow}, nil)
			},
			wantResult: &referrer.ListResult{
				Manifests:     []ocispec.Descriptor{fixtureDescriptor},
				FilterApplied: false,
			},
		},
		{
			name:         "happy path: artifact type filter returns matching referrers",
			namespace:    testNamespace,
			subject:      subjectDigest,
			artifactType: testArtifactType,
			setup: func(m *mockmeta.MetadataStore) {
				m.On("NamespaceExists", mock.Anything, testNamespace).Return(true, nil)
				m.On("Atomic", mock.Anything, mock.Anything).Return(nil)
				m.Tx.On("ListReferrers", mock.Anything, testNamespace, subjectDigest).
					Return([]metastore.ReferrerRow{fixtureRow}, nil)
			},
			wantResult: &referrer.ListResult{
				Manifests:     []ocispec.Descriptor{fixtureDescriptor},
				FilterApplied: true,
			},
		},
		{
			name:         "happy path: artifact type filter excludes non-matching referrers",
			namespace:    testNamespace,
			subject:      subjectDigest,
			artifactType: "application/vnd.other",
			setup: func(m *mockmeta.MetadataStore) {
				m.On("NamespaceExists", mock.Anything, testNamespace).Return(true, nil)
				m.On("Atomic", mock.Anything, mock.Anything).Return(nil)
				m.Tx.On("ListReferrers", mock.Anything, testNamespace, subjectDigest).
					Return([]metastore.ReferrerRow{fixtureRow}, nil)
			},
			wantResult: &referrer.ListResult{
				Manifests:     []ocispec.Descriptor{},
				FilterApplied: true,
			},
		},
		{
			name:      "happy path: empty referrer list",
			namespace: testNamespace,
			subject:   subjectDigest,
			setup: func(m *mockmeta.MetadataStore) {
				m.On("NamespaceExists", mock.Anything, testNamespace).Return(true, nil)
				m.On("Atomic", mock.Anything, mock.Anything).Return(nil)
				m.Tx.On("ListReferrers", mock.Anything, testNamespace, subjectDigest).
					Return([]metastore.ReferrerRow{}, nil)
			},
			wantResult: &referrer.ListResult{
				Manifests:     []ocispec.Descriptor{},
				FilterApplied: false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, ms := newTestSvc(t)

			if tc.setup != nil {
				tc.setup(ms)
			}

			result, err := svc.ListReferrers(t.Context(), tc.namespace, tc.subject, tc.artifactType)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)

			if tc.wantResult != nil {
				assert.Equal(t, *tc.wantResult, result)
			}
		})
	}
}
