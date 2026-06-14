package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/handler"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/middleware"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/response"
)

func New(logger *zap.Logger, healthHandler *handler.HealthHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.TenantContext)
	r.Use(chimiddleware.StripSlashes)

	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		response.NotFound(w, req)
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, req *http.Request) {
		response.Error(w, req, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed", nil)
	})

	r.Get("/health", healthHandler.Live)
	r.Get("/ready", healthHandler.Ready)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/meta", metaHandler)
	})

	return r
}

func metaHandler(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, r, http.StatusOK, map[string]any{
		"version": "1.0.0",
		"name":    "unified-trust-platform",
		"features": []string{
			"oauth2", "oidc", "mfa", "rbac", "abac", "pbac",
			"delegation", "jit", "federation", "audit",
		},
	})
}
