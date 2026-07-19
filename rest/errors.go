package rest

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/tiamiru/omnistash/internal/logtag"
	"github.com/tiamiru/omnistash/internal/ocierror"
)

var (
	errRangeHeaderInvalid        = errors.New("invalid Range header")
	errContentRangeHeaderInvalid = errors.New("invalid Content-Range header")
)

const (
	// ociCodeNameExists is returned when attempting to
	// create a namespace that already exists.
	ociCodeNameExists = "NAME_EXISTS"

	ociCodeNameInvalid       = "NAME_INVALID"
	ociCodeNameUnknown       = "NAME_UNKNOWN"
	ociCodeInternalError     = "INTERNAL_ERROR"
	ociCodeBlobUnknown       = "BLOB_UNKNOWN"
	ociCodeBlobUploadUnknown = "BLOB_UPLOAD_UNKNOWN"
	ociCodeBlobUploadInvalid = "BLOB_UPLOAD_INVALID"
	ociCodeDigestInvalid     = "DIGEST_INVALID"
	ociCodeSizeInvalid       = "SIZE_INVALID"
	ociCodeUnsupported       = "UNSUPPORTED"
)

type ociErrorEntry struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ociErrorEnvelope struct {
	Errors []ociErrorEntry `json:"errors"`
}

func (h *RegistryHandler) writeOCIError(w http.ResponseWriter, caller string, status int, code, message string) {
	body, err := json.Marshal(ociErrorEnvelope{Errors: []ociErrorEntry{{Code: code, Message: message}}})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, writeErr := w.Write(body)
	if writeErr != nil {
		h.logger.Warn("writeOCIError: write error response", logtag.Err(writeErr), slog.String("caller", caller))
	}
}

func (h *RegistryHandler) registryErrToHTTP(w http.ResponseWriter, caller string, err error) {
	status, code := errToStatusCode(err)

	var clientMsg string
	if status >= http.StatusInternalServerError {
		h.logger.Error(caller, logtag.Err(err))
		clientMsg = "internal server error"
	} else {
		h.logger.Warn(caller, logtag.Err(err))
		clientMsg = err.Error()
	}

	h.writeOCIError(w, caller, status, code, clientMsg)
}

func (h *RegistryHandler) registryErrToHTTPNoBody(w http.ResponseWriter, caller string, err error) {
	status, _ := errToStatusCode(err)

	if status >= http.StatusInternalServerError {
		h.logger.Error(caller, logtag.Err(err))
	} else {
		h.logger.Warn(caller, logtag.Err(err))
	}

	w.WriteHeader(status)
}

func errToStatusCode(err error) (int, string) {
	switch {
	case errors.Is(err, ocierror.ErrNameExists):
		return http.StatusConflict, ociCodeNameExists
	case errors.Is(err, ocierror.ErrNameInvalid):
		return http.StatusBadRequest, ociCodeNameInvalid
	case errors.Is(err, ocierror.ErrNameUnknown):
		return http.StatusNotFound, ociCodeNameUnknown
	case errors.Is(err, ocierror.ErrDigestInvalid):
		return http.StatusBadRequest, ociCodeDigestInvalid
	case errors.Is(err, ocierror.ErrSizeInvalid):
		return http.StatusBadRequest, ociCodeSizeInvalid
	case errors.Is(err, ocierror.ErrUnsupported):
		return http.StatusBadRequest, ociCodeUnsupported
	case errors.Is(err, ocierror.ErrBlobUnknown):
		return http.StatusNotFound, ociCodeBlobUnknown
	case errors.Is(err, ocierror.ErrBlobUploadUnknown):
		return http.StatusNotFound, ociCodeBlobUploadUnknown
	case errors.Is(err, ocierror.ErrRangeNotSatisfiable):
		return http.StatusRequestedRangeNotSatisfiable, ociCodeBlobUploadInvalid
	case errors.Is(err, errRangeHeaderInvalid):
		return http.StatusBadRequest, ociCodeBlobUploadInvalid
	case errors.Is(err, errContentRangeHeaderInvalid):
		return http.StatusBadRequest, ociCodeBlobUploadInvalid
	default:
		return http.StatusInternalServerError, ociCodeInternalError
	}
}
