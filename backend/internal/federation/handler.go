package federation

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/middleware"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/response"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/validate"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ProviderRoutes(r chi.Router) {
	r.Post("/", h.CreateProvider)
	r.Get("/", h.ListProviders)
	r.Get("/{id}", h.GetProvider)
	r.Delete("/{id}", h.DeleteProvider)
	r.Get("/{id}/login", h.InitiateLogin)
	r.Get("/{id}/callback", h.Callback)
}

func (h *Handler) CreateProvider(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}

	var body struct {
		Name          string   `json:"name"`
		Protocol      string   `json:"protocol"`
		ClientID      string   `json:"clientId"`
		ClientSecret  string   `json:"clientSecret"`
		DiscoveryURL  string   `json:"discoveryUrl"`
		Scopes        []string `json:"scopes"`
		AutoProvision bool     `json:"autoProvision"`
	}
	if err := validate.DecodeJSON(r, &body); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}

	errs := validate.FieldErrors{}
	validate.Required("name", body.Name, errs)
	validate.Required("clientId", body.ClientID, errs)
	validate.Required("clientSecret", body.ClientSecret, errs)
	if body.Protocol == "" {
		body.Protocol = "oidc"
	}
	if len(errs) > 0 {
		response.ValidationFailed(w, r, errs)
		return
	}

	p, err := h.svc.CreateProvider(r.Context(), tenantID, body.Name, body.Protocol, body.ClientID, body.ClientSecret, body.DiscoveryURL, body.Scopes, body.AutoProvision)
	if err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}
	response.JSON(w, r, http.StatusCreated, p)
}

func (h *Handler) ListProviders(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}
	providers, err := h.svc.ListProviders(r.Context(), tenantID)
	if err != nil {
		response.InternalError(w, r)
		return
	}
	response.Collection(w, r, http.StatusOK, providers, nil)
}

func (h *Handler) GetProvider(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "invalid provider id")
		return
	}
	p, err := h.svc.GetProvider(r.Context(), tenantID, id)
	if err != nil {
		response.InternalError(w, r)
		return
	}
	if p == nil {
		response.NotFound(w, r)
		return
	}
	response.JSON(w, r, http.StatusOK, p)
}

func (h *Handler) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "invalid provider id")
		return
	}
	if err := h.svc.DeleteProvider(r.Context(), tenantID, id); err != nil {
		response.InternalError(w, r)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// InitiateLogin redirects the user to the external IdP authorization endpoint.
func (h *Handler) InitiateLogin(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "invalid provider id")
		return
	}

	redirectURI := r.URL.Query().Get("redirect_uri")
	if redirectURI == "" {
		response.BadRequest(w, r, "redirect_uri required")
		return
	}

	authURL, state, err := h.svc.BuildAuthURL(r.Context(), tenantID, id, redirectURI)
	if err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}

	// Return JSON with authUrl and state for the client to store and redirect
	response.JSON(w, r, http.StatusOK, map[string]string{
		"authorizationUrl": authURL,
		"state":            state,
	})
}

// Callback handles the redirect from the external IdP after authentication.
func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "invalid provider id")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		response.BadRequest(w, r, "code is required")
		return
	}
	redirectURI := r.URL.Query().Get("redirect_uri")
	if redirectURI == "" {
		response.BadRequest(w, r, "redirect_uri is required")
		return
	}

	identity, err := h.svc.HandleCallback(r.Context(), tenantID, id, code, redirectURI)
	if err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}
	response.JSON(w, r, http.StatusOK, identity)
}
