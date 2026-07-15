package rest

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/tiamiru/omnistash/internal/logtag"
	"github.com/tiamiru/omnistash/internal/namespace"
)

const (
	// ociCodeNameExists is returned when attempting to
	// create a namespace that already exists.
	ociCodeNameExists = "NAME_EXISTS"

	ociCodeNameInvalid = "NAME_INVALID"
	ociCodeNameUnknown = "NAME_UNKNOWN"
	ociCodeUnsupported = "UNSUPPORTED"
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
	var status int
	var code string

	switch {
	case errors.Is(err, namespace.ErrNameExists):
		status, code = http.StatusConflict, ociCodeNameExists
	case errors.Is(err, namespace.ErrNameInvalid):
		status, code = http.StatusBadRequest, ociCodeNameInvalid
	case errors.Is(err, namespace.ErrNameUnknown):
		status, code = http.StatusNotFound, ociCodeNameUnknown
	default:
		status, code = http.StatusInternalServerError, ociCodeUnsupported
	}

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
