package rest_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tiamiru/omnistash/internal/namespace"
	"github.com/tiamiru/omnistash/rest"
)

var errUnexpected = errors.New("unexpected failure")

type testErrorEntry struct {
	Code string `json:"code"`
}

type testErrorEnvelope struct {
	Errors []testErrorEntry `json:"errors"`
}

type stubNamespaceService struct {
	createErr error
	deleteErr error
}

func (s *stubNamespaceService) CreateNamespace(_ context.Context, _ string) (namespace.Namespace, error) {
	return namespace.Namespace{}, s.createErr
}

func (s *stubNamespaceService) DeleteNamespace(_ context.Context, _ string) (namespace.Namespace, error) {
	return namespace.Namespace{}, s.deleteErr
}

func newTestHandler(svc rest.NamespaceService) *rest.RegistryHandler {
	return rest.NewRegistryHandler(
		slog.New(slog.DiscardHandler),
		svc,
		"test", "abc123", "2026-07-15",
	)
}

func TestRegistryHandler_registryErrToHTTP(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "happy path: converts ErrNameInvalid to 400 NAME_INVALID",
			err:        namespace.ErrNameInvalid,
			wantStatus: http.StatusBadRequest,
			wantCode:   "NAME_INVALID",
		},
		{
			name:       "happy path: converts ErrNameExists to 409 NAME_EXISTS",
			err:        namespace.ErrNameExists,
			wantStatus: http.StatusConflict,
			wantCode:   "NAME_EXISTS",
		},
		{
			name:       "happy path: converts ErrNameUnknown to 404 NAME_UNKNOWN",
			err:        namespace.ErrNameUnknown,
			wantStatus: http.StatusNotFound,
			wantCode:   "NAME_UNKNOWN",
		},
		{
			name:       "happy path: converts unknown error to 500 UNSUPPORTED",
			err:        errUnexpected,
			wantStatus: http.StatusInternalServerError,
			wantCode:   "UNSUPPORTED",
		},
		{
			name:       "happy path: converts wrapped sentinel to 409 NAME_EXISTS",
			err:        fmt.Errorf("Service.CreateNamespace: %w", namespace.ErrNameExists),
			wantStatus: http.StatusConflict,
			wantCode:   "NAME_EXISTS",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler := newTestHandler(&stubNamespaceService{createErr: tc.err})

			req := httptest.NewRequestWithContext(
				t.Context(),
				http.MethodPost,
				"/v2/namespaces",
				strings.NewReader(`{"name":"myrepo"}`),
			)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.HandleCreateNamespace(w, req)

			res := w.Result()
			assert.Equal(t, tc.wantStatus, res.StatusCode)
			assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

			var envelope testErrorEnvelope
			err := json.NewDecoder(res.Body).Decode(&envelope)
			require.NoError(t, err)
			require.Len(t, envelope.Errors, 1)
			assert.Equal(t, tc.wantCode, envelope.Errors[0].Code)
		})
	}
}
