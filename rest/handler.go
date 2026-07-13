package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/tiamiru/omnistash/internal/health"
	"github.com/tiamiru/omnistash/internal/logtag"
)

type RegistryHandler struct {
	health *health.Checker
	logger *slog.Logger
}

func NewRegistryHandler(logger *slog.Logger, version string) *RegistryHandler {
	if logger == nil {
		panic("rest.NewRegistryHandler: logger must not be nil")
	}

	return &RegistryHandler{
		health: health.NewChecker(version),
		logger: logger,
	}
}

// HandleHealth implements GET /health.
// It returns the server's status and build version.
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
