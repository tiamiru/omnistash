package rest

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"

	"github.com/tiamiru/omnistash/internal/testutil/stub"
)

func newManifestHandler(svc *stub.ManifestService) *RegistryHandler {
	return &RegistryHandler{
		manifest: svc,
		logger:   slog.New(slog.DiscardHandler),
	}
}

func newManifestRequest(ctx context.Context, method, rawQuery string, body []byte) *http.Request {
	ref := stub.FixtureDigest.String()
	r := httptest.NewRequestWithContext(
		ctx, method, "/manifests/"+ref+"?"+rawQuery, bytes.NewReader(body),
	)
	r = withRepoName(r, "myrepo")
	r.SetPathValue("reference", ref)

	return r
}

func TestHandleGetManifest(t *testing.T) {
	t.Parallel()

	svc := stub.NewManifestService()
	h := newManifestHandler(svc)

	w := httptest.NewRecorder()
	h.handleGetManifest(w, newManifestRequest(t.Context(), http.MethodGet, "", nil))

	res := w.Result()
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, stub.FixtureMediaType, res.Header.Get("Content-Type"))
	assert.Equal(t, "512", res.Header.Get("Content-Length"))
	assert.Equal(t, stub.FixtureDigest.String(), res.Header.Get("Docker-Content-Digest"))
	assert.Equal(t, stub.FixtureBody(), w.Body.Bytes())
	assert.Equal(t, []string{"GetManifest"}, svc.Calls)
}

func TestHandleHeadManifest(t *testing.T) {
	t.Parallel()

	svc := stub.NewManifestService()
	h := newManifestHandler(svc)

	w := httptest.NewRecorder()
	h.handleHeadManifest(w, newManifestRequest(t.Context(), http.MethodHead, "", nil))

	res := w.Result()
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, stub.FixtureMediaType, res.Header.Get("Content-Type"))
	assert.Equal(t, "512", res.Header.Get("Content-Length"))
	assert.Equal(t, stub.FixtureDigest.String(), res.Header.Get("Docker-Content-Digest"))
	assert.Empty(t, w.Body.Bytes())
	assert.Equal(t, []string{"HeadManifest"}, svc.Calls)
}

func TestHandlePutManifest(t *testing.T) {
	t.Parallel()

	subjectDigest := digest.NewDigestFromEncoded(
		digest.SHA256,
		"b5bb9d8014a0f9b1d61e21e796d78dccdf1352f23cd32812f4850b878ae4944c",
	)

	testCases := []struct {
		name        string
		body        []byte
		queryParams string
		mutateStub  func(*stub.ManifestService)
		wantStatus  int
		wantCalls   []string
		wantSubject string
	}{
		{
			name:       "error path: body exceeds maxManifestSize returns 413",
			body:       bytes.Repeat([]byte("x"), maxManifestSize+1),
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:       "happy path: PutManifest",
			body:       []byte(`{}`),
			wantStatus: http.StatusCreated,
			wantCalls:  []string{"PutManifest"},
		},
		{
			name: "happy path: non-nil subject sets OCI-Subject header",
			body: []byte(`{}`),
			mutateStub: func(s *stub.ManifestService) {
				s.Subject = &subjectDigest
			},
			wantStatus:  http.StatusCreated,
			wantCalls:   []string{"PutManifest"},
			wantSubject: subjectDigest.String(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := stub.NewManifestService()
			if tc.mutateStub != nil {
				tc.mutateStub(svc)
			}

			h := newManifestHandler(svc)

			w := httptest.NewRecorder()
			h.handlePutManifest(w, newManifestRequest(t.Context(), http.MethodPut, tc.queryParams, tc.body))

			res := w.Result()
			assert.Equal(t, tc.wantStatus, res.StatusCode)
			assert.Equal(t, tc.wantCalls, svc.Calls)

			if tc.wantStatus != http.StatusCreated {
				return
			}

			assert.Equal(t, stub.FixtureLocation, res.Header.Get("Location"))
			assert.Equal(t, stub.FixtureDigest.String(), res.Header.Get("Docker-Content-Digest"))
			assert.Equal(t, tc.wantSubject, res.Header.Get(headerOCISubject))
		})
	}
}

func newManifestRequestWithRef(ctx context.Context, method, reference string) *http.Request {
	r := httptest.NewRequestWithContext(ctx, method, "/manifests/"+reference, http.NoBody)
	r = withRepoName(r, "myrepo")
	r.SetPathValue("reference", reference)

	return r
}

func TestHandleDeleteManifest(t *testing.T) {
	t.Parallel()

	svc := stub.NewManifestService()
	h := newManifestHandler(svc)

	w := httptest.NewRecorder()
	h.handleDeleteManifest(w, newManifestRequestWithRef(t.Context(), http.MethodDelete, stub.FixtureDigest.String()))

	assert.Equal(t, http.StatusAccepted, w.Result().StatusCode)
	assert.Equal(t, []string{"DeleteManifest"}, svc.Calls)
}
