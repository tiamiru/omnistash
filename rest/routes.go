package rest

import "net/http"

func setupRoutes(mux *http.ServeMux, h *RegistryHandler) {
	mux.HandleFunc("GET /health", h.HandleHealth)

	mux.HandleFunc("POST /v2/namespaces", h.HandleCreateNamespace)
	mux.HandleFunc("DELETE /v2/namespaces/{namespace...}", h.HandleDeleteNamespace)
}
