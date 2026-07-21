package rest

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	specs "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/testutil/stub"
)

func newReferrersHandler(svc *stub.ReferrersService) *RegistryHandler {
	return &RegistryHandler{
		referrers: svc,
		logger:    slog.New(slog.DiscardHandler),
	}
}

func newReferrersRequest(ctx context.Context, d string, artifactType string) *http.Request {
	url := "/referrers/" + d
	if artifactType != "" {
		url += "?artifactType=" + artifactType
	}

	r := httptest.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	r = withRepoName(r, "myrepo")
	r.SetPathValue("digest", d)

	return r
}

func TestHandleGetReferrers(t *testing.T) {
	t.Parallel()

	type wantResponse struct {
		status       int
		filterHeader string
		calls        []string
		body         ocispec.Index
	}

	fixtureManifest := ocispec.Descriptor{
		MediaType:    stub.FixtureMediaType,
		Digest:       stub.FixtureDigest,
		Size:         stub.FixtureSizeBytes,
		ArtifactType: stub.FixtureArtifactType,
	}

	testCases := []struct {
		name         string
		digest       string
		artifactType string
		setup        func(*stub.ReferrersService)
		wantResponse wantResponse
	}{
		{
			name:   "error path: invalid digest",
			digest: "notadigest",
			wantResponse: wantResponse{
				status: http.StatusBadRequest,
			},
		},
		{
			name:   "happy path: returns manifests",
			digest: stub.FixtureDigest.String(),
			wantResponse: wantResponse{
				status: http.StatusOK,
				calls:  []string{"ListReferrers"},
				body: ocispec.Index{
					Versioned: specs.Versioned{SchemaVersion: ociImageIndexSchemaVersion},
					MediaType: ocispec.MediaTypeImageIndex,
					Manifests: []ocispec.Descriptor{fixtureManifest},
				},
			},
		},
		{
			name:         "happy path: filter applied sets header",
			digest:       stub.FixtureDigest.String(),
			artifactType: "application/vnd.example.test",
			setup: func(svc *stub.ReferrersService) {
				svc.FilterApplied = true
			},
			wantResponse: wantResponse{
				status:       http.StatusOK,
				filterHeader: "artifactType",
				calls:        []string{"ListReferrers"},
				body: ocispec.Index{
					Versioned: specs.Versioned{SchemaVersion: ociImageIndexSchemaVersion},
					MediaType: ocispec.MediaTypeImageIndex,
					Manifests: []ocispec.Descriptor{fixtureManifest},
				},
			},
		},
		{
			name:   "happy path: empty manifests serializes as array",
			digest: stub.FixtureDigest.String(),
			setup: func(svc *stub.ReferrersService) {
				svc.Manifests = []ocispec.Descriptor{}
			},
			wantResponse: wantResponse{
				status: http.StatusOK,
				calls:  []string{"ListReferrers"},
				body: ocispec.Index{
					Versioned: specs.Versioned{SchemaVersion: ociImageIndexSchemaVersion},
					MediaType: ocispec.MediaTypeImageIndex,
					Manifests: []ocispec.Descriptor{},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := stub.NewReferrersService()
			if tc.setup != nil {
				tc.setup(svc)
			}
			h := newReferrersHandler(svc)

			w := httptest.NewRecorder()
			h.handleGetReferrers(w, newReferrersRequest(t.Context(), tc.digest, tc.artifactType))

			res := w.Result()
			assert.Equal(t, tc.wantResponse.status, res.StatusCode)
			assert.Equal(t, tc.wantResponse.calls, svc.Calls)

			if tc.wantResponse.status != http.StatusOK {
				return
			}

			assert.Equal(t, ocispec.MediaTypeImageIndex, res.Header.Get("Content-Type"))
			assert.Equal(t, tc.wantResponse.filterHeader, res.Header.Get(headerOCIFiltersApplied))

			var got ocispec.Index
			err := json.NewDecoder(w.Body).Decode(&got)
			require.NoError(t, err)
			assert.Equal(t, tc.wantResponse.body, got)
		})
	}
}
