package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/tiamiru/omnistash/internal/health"
	"github.com/tiamiru/omnistash/internal/logtag"
)

const maxNamespaceRequestBytes = 4 * 1024

type RegistryHandler struct {
	health    *health.Checker
	namespace NamespaceService
	logger    *slog.Logger
}

func NewRegistryHandler(logger *slog.Logger, ns NamespaceService, version, commit, date string) *RegistryHandler {
	if logger == nil {
		panic("NewRegistryHandler: logger must not be nil")
	}

	return &RegistryHandler{
		health:    health.NewChecker(version, commit, date),
		namespace: ns,
		logger:    logger,
	}
}

// HandleHealth implements GET /health.
func (h *RegistryHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	status := h.health.Check()

	body, err := json.Marshal(status)
	if err != nil {
		h.logger.Warn("HandleHealth: marshal response", logtag.Err(err))
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	_, writeErr := w.Write(body)
	if writeErr != nil {
		h.logger.Warn("HandleHealth: write response", logtag.Err(writeErr))
	}
}

// HandleCreateNamespace implements POST /v2/namespaces.
func (h *RegistryHandler) HandleCreateNamespace(w http.ResponseWriter, r *http.Request) {
	var body createNamespaceRequest

	r.Body = http.MaxBytesReader(w, r.Body, maxNamespaceRequestBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(&body)
	if err != nil {
		h.writeOCIError(w, "HandleCreateNamespace", http.StatusBadRequest, ociCodeNameInvalid, "invalid request body")

		return
	}

	ns, err := h.namespace.CreateNamespace(r.Context(), body.Name)
	if err != nil {
		h.registryErrToHTTP(w, "HandleCreateNamespace", err)

		return
	}

	resp, err := json.Marshal(createNamespaceResponse{
		Name:      ns.Name,
		CreatedAt: ns.CreatedAt,
		UpdatedAt: ns.UpdatedAt,
	})
	if err != nil {
		h.logger.Warn("HandleCreateNamespace: marshal response", logtag.Err(err))
		h.writeOCIError(
			w,
			"HandleCreateNamespace",
			http.StatusInternalServerError,
			ociCodeUnsupported,
			"internal server error",
		)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	_, writeErr := w.Write(resp)
	if writeErr != nil {
		h.logger.Warn("HandleCreateNamespace: write response", logtag.Err(writeErr))
	}
}

// HandleDeleteNamespace implements DELETE /v2/namespaces/{namespace...}.
func (h *RegistryHandler) HandleDeleteNamespace(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("namespace")

	ns, err := h.namespace.DeleteNamespace(r.Context(), name)
	if err != nil {
		h.registryErrToHTTP(w, "HandleDeleteNamespace", err)

		return
	}

	resp, err := json.Marshal(deleteNamespaceResponse{
		Name:      ns.Name,
		CreatedAt: ns.CreatedAt,
		UpdatedAt: ns.UpdatedAt,
	})
	if err != nil {
		h.logger.Warn("HandleDeleteNamespace: marshal response", logtag.Err(err))
		h.writeOCIError(
			w,
			"HandleDeleteNamespace",
			http.StatusInternalServerError,
			ociCodeUnsupported,
			"internal server error",
		)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	_, writeErr := w.Write(resp)
	if writeErr != nil {
		h.logger.Warn("HandleDeleteNamespace: write response", logtag.Err(writeErr))
	}
}
