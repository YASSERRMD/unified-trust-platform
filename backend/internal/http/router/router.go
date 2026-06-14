package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/handler"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/middleware"
)

func New(healthHandler *handler.HealthHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(chimiddleware.StripSlashes)

	r.Get("/health", healthHandler.Live)
	r.Get("/ready", healthHandler.Ready)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/meta", func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"version":"1.0.0","name":"unified-trust-platform"}`))
		})
	})

	return r
}
