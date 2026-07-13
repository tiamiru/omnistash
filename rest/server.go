package rest

import (
	"net/http"
	"time"
)

const readHeaderTimeout = 10 * time.Second

// NewServer returns an *http.Server with all registered routes.
func NewServer(h *RegistryHandler, addr string) *http.Server {
	if h == nil {
		panic("server.NewServer: h must not be nil")
	}

	mux := http.NewServeMux()
	setupRoutes(mux, h)

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}
}
