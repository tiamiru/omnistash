package rest

import "net/http"

func setupRoutes(mux *http.ServeMux, h *RegistryHandler) {
	mux.HandleFunc("GET /health", h.HandleHealth)
}
