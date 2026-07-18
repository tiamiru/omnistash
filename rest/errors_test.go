package rest

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tiamiru/omnistash/internal/blob"
	"github.com/tiamiru/omnistash/internal/namespace"
)

var errUnexpected = errors.New("unexpected failure")

func TestErrToStatusCode(t *testing.T) {
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
			wantCode:   ociCodeNameInvalid,
		},
		{
			name:       "happy path: converts ErrDigestInvalid to 400 DIGEST_INVALID",
			err:        blob.ErrDigestInvalid,
			wantStatus: http.StatusBadRequest,
			wantCode:   ociCodeDigestInvalid,
		},
		{
			name:       "happy path: converts ErrSizeInvalid to 400 SIZE_INVALID",
			err:        blob.ErrSizeInvalid,
			wantStatus: http.StatusBadRequest,
			wantCode:   ociCodeSizeInvalid,
		},
		{
			name:       "happy path: converts ErrNameUnknown to 404 NAME_UNKNOWN",
			err:        namespace.ErrNameUnknown,
			wantStatus: http.StatusNotFound,
			wantCode:   ociCodeNameUnknown,
		},
		{
			name:       "happy path: converts ErrBlobUnknown to 404 BLOB_UNKNOWN",
			err:        blob.ErrBlobUnknown,
			wantStatus: http.StatusNotFound,
			wantCode:   ociCodeBlobUnknown,
		},
		{
			name:       "happy path: converts wrapped ErrBlobUnknown to 404 BLOB_UNKNOWN",
			err:        fmt.Errorf("Service.DeleteBlob: %w", blob.ErrBlobUnknown),
			wantStatus: http.StatusNotFound,
			wantCode:   ociCodeBlobUnknown,
		},
		{
			name:       "happy path: converts ErrBlobUploadUnknown to 404 BLOB_UPLOAD_UNKNOWN",
			err:        blob.ErrBlobUploadUnknown,
			wantStatus: http.StatusNotFound,
			wantCode:   ociCodeBlobUploadUnknown,
		},
		{
			name:       "happy path: converts ErrNameExists to 409 NAME_EXISTS",
			err:        namespace.ErrNameExists,
			wantStatus: http.StatusConflict,
			wantCode:   ociCodeNameExists,
		},
		{
			name:       "happy path: converts wrapped ErrNameExists to 409 NAME_EXISTS",
			err:        fmt.Errorf("Service.CreateNamespace: %w", namespace.ErrNameExists),
			wantStatus: http.StatusConflict,
			wantCode:   ociCodeNameExists,
		},
		{
			name:       "happy path: converts unknown error to 500 INTERNAL_ERROR",
			err:        errUnexpected,
			wantStatus: http.StatusInternalServerError,
			wantCode:   ociCodeInternalError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotStatus, gotCode := errToStatusCode(tc.err)
			assert.Equal(t, tc.wantStatus, gotStatus)
			assert.Equal(t, tc.wantCode, gotCode)
		})
	}
}
