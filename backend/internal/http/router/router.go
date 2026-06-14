package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	auditpkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/audit"
	delegpkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/delegation"
	fedpkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/federation"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/handler"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/middleware"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/response"
	jitpkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/jit"
	mfapkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/mfa"
	oauthpkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/oauth"
	policypkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/policy"
	tenantpkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/tenant"
	userpkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/user"
)

type Handlers struct {
	Health     *handler.HealthHandler
	Tenant     *tenantpkg.Handler
	User       *userpkg.Handler
	OAuth      *oauthpkg.Handler
	MFA        *mfapkg.Handler
	RBAC       *policypkg.RBACHandler
	Policy     *policypkg.PolicyHandler
	Delegation *delegpkg.Handler
	JIT        *jitpkg.Handler
	Federation *fedpkg.Handler
	Audit      *auditpkg.Handler
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
		if h.MFA != nil {
			r.Route("/mfa", h.MFA.Routes)
		}
		if h.RBAC != nil {
			r.Route("/roles", h.RBAC.RoleRoutes)
			r.Route("/permissions", h.RBAC.PermissionRoutes)
			r.Route("/users/{userId}/roles", h.RBAC.UserRoleRoutes)
		}
		if h.Policy != nil {
			r.Route("/policies", h.Policy.PolicyRoutes)
		}
		if h.Delegation != nil {
			r.Route("/delegations", h.Delegation.Routes)
		}
		if h.JIT != nil {
			r.Route("/jit/requests", h.JIT.Routes)
			r.Route("/jit/grants", h.JIT.GrantRoutes)
		}
		if h.Federation != nil {
			r.Route("/federation/providers", h.Federation.ProviderRoutes)
		}
		if h.Audit != nil {
			r.Route("/audit/events", h.Audit.Routes)
		}
		r.Route("/authz", func(r chi.Router) {
			if h.Policy != nil {
				h.Policy.EvaluateRoute(r)
			} else if h.RBAC != nil {
				h.RBAC.EvaluateRoute(r)
			}
		})
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
