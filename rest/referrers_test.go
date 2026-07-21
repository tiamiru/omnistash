package rest

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/referrer"
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

	svc := stub.NewReferrersService()
	h := newReferrersHandler(svc)

	w := httptest.NewRecorder()
	h.handleGetReferrers(w, newReferrersRequest(t.Context(), stub.FixtureDigest.String(), ""))

	res := w.Result()
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, mediaTypeOCIImageIndex, res.Header.Get("Content-Type"))
	assert.Empty(t, res.Header.Get(headerOCIFiltersApplied))
	assert.Equal(t, []string{"ListReferrers"}, svc.Calls)

	var index ociImageIndex
	err := json.NewDecoder(w.Body).Decode(&index)
	require.NoError(t, err)
	assert.Equal(t, 2, index.SchemaVersion)
	assert.Equal(t, mediaTypeOCIImageIndex, index.MediaType)
	assert.Len(t, index.Manifests, 1)
}

func TestHandleGetReferrersFilterApplied(t *testing.T) {
	t.Parallel()

	svc := stub.NewReferrersService()
	svc.FilterApplied = true
	h := newReferrersHandler(svc)

	w := httptest.NewRecorder()
	h.handleGetReferrers(
		w,
		newReferrersRequest(t.Context(), stub.FixtureDigest.String(), "application/vnd.example.test"),
	)

	res := w.Result()
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "artifactType", res.Header.Get(headerOCIFiltersApplied))
	assert.Equal(t, []string{"ListReferrers"}, svc.Calls)
}

func TestHandleGetReferrersInvalidDigest(t *testing.T) {
	t.Parallel()

	svc := stub.NewReferrersService()
	h := newReferrersHandler(svc)

	w := httptest.NewRecorder()
	h.handleGetReferrers(w, newReferrersRequest(t.Context(), "notadigest", ""))

	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	assert.Empty(t, svc.Calls)
}

func TestHandleGetReferrersEmptyManifests(t *testing.T) {
	t.Parallel()

	svc := stub.NewReferrersService()
	svc.Manifests = []referrer.Descriptor{}
	h := newReferrersHandler(svc)

	w := httptest.NewRecorder()
	h.handleGetReferrers(w, newReferrersRequest(t.Context(), stub.FixtureDigest.String(), ""))

	res := w.Result()
	assert.Equal(t, http.StatusOK, res.StatusCode)

	var index ociImageIndex
	err := json.NewDecoder(w.Body).Decode(&index)
	require.NoError(t, err)
	assert.Empty(t, index.Manifests)
}
