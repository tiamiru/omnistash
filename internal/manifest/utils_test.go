package manifest

import (
	"testing"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/ocierror"
)

func TestParseManifestBody(t *testing.T) {
	t.Parallel()

	validBody := []byte(`{"mediaType":"application/vnd.oci.image.manifest.v1+json"}`)
	bodyWithSubject := []byte(
		`{"mediaType":"application/vnd.oci.image.manifest.v1+json",` +
			`"subject":{"digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}`,
	)
	subjectDigest := digest.Digest("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	testCases := []struct {
		name        string
		contentType string
		body        []byte
		wantErr     error
		wantResult  *ocispec.Manifest
	}{
		{
			name:    "error path: invalid json",
			body:    []byte(`not json`),
			wantErr: ocierror.ErrManifestInvalid,
		},
		{
			name:    "error path: missing media type in body and header",
			body:    []byte(`{}`),
			wantErr: ocierror.ErrManifestInvalid,
		},
		{
			name:        "happy path: media type from content-type header",
			contentType: "application/vnd.oci.image.manifest.v1+json",
			body:        []byte(`{}`),
			wantResult:  &ocispec.Manifest{MediaType: "application/vnd.oci.image.manifest.v1+json"},
		},
		{
			name:       "happy path: media type from body",
			body:       validBody,
			wantResult: &ocispec.Manifest{MediaType: "application/vnd.oci.image.manifest.v1+json"},
		},
		{
			name:        "happy path: content-type header overrides body media type",
			contentType: "application/vnd.docker.distribution.manifest.v2+json",
			body:        validBody,
			wantResult:  &ocispec.Manifest{MediaType: "application/vnd.docker.distribution.manifest.v2+json"},
		},
		{
			name: "happy path: subject digest extracted",
			body: bodyWithSubject,
			wantResult: &ocispec.Manifest{
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Subject:   &ocispec.Descriptor{Digest: subjectDigest},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseManifestBody(tc.contentType, tc.body)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, *tc.wantResult, got)
		})
	}
}
