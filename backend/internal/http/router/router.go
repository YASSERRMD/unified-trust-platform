package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/handler"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/middleware"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/response"
	oauthpkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/oauth"
	tenantpkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/tenant"
	userpkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/user"
)

type Handlers struct {
	Health *handler.HealthHandler
	Tenant *tenantpkg.Handler
	User   *userpkg.Handler
	OAuth  *oauthpkg.Handler
}

func New(logger *zap.Logger, h *Handlers) http.Handler {
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

	r.Get("/health", h.Health.Live)
	r.Get("/ready", h.Health.Ready)

	if h.OAuth != nil {
		r.Get("/.well-known/openid-configuration", h.OAuth.Discovery)
		r.Get("/.well-known/jwks.json", h.OAuth.JWKS)
		r.Get("/oauth2/authorize", h.OAuth.Authorize)
		r.Post("/oauth2/token", h.OAuth.Token)
		r.Post("/oauth2/revoke", h.OAuth.Revoke)
		r.Get("/oauth2/userinfo", h.OAuth.UserInfo)
		r.Post("/oauth2/logout", h.OAuth.Logout)
	}

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/meta", metaHandler)

		if h.Tenant != nil {
			r.Route("/tenants", h.Tenant.Routes)
		}
		if h.User != nil {
			r.Route("/users", h.User.Routes)
		}
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
