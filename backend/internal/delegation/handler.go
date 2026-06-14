package delegation

import (
	"net/http"
	"time"

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

func (h *Handler) Routes(r chi.Router) {
	r.Post("/", h.Create)
	r.Get("/", h.ListMine)
	r.Get("/{id}", h.GetByID)
	r.Delete("/{id}", h.Revoke)
	r.Get("/effective", h.Effective)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}

	var body struct {
		DelegatorID  string   `json:"delegatorId"`
		DelegateID   string   `json:"delegateId"`
		ScopeType    string   `json:"scopeType"`
		GrantedRoles []string `json:"grantedRoles"`
		Reason       string   `json:"reason"`
		StartsAt     *string  `json:"startsAt"`
		ExpiresAt    *string  `json:"expiresAt"`
	}
	if err := validate.DecodeJSON(r, &body); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}

	errs := validate.FieldErrors{}
	validate.Required("delegatorId", body.DelegatorID, errs)
	validate.Required("delegateId", body.DelegateID, errs)
	if len(body.GrantedRoles) == 0 {
		errs["grantedRoles"] = "at least one role required"
	}
	if len(errs) > 0 {
		response.ValidationFailed(w, r, errs)
		return
	}

	delegatorID, err := uuid.Parse(body.DelegatorID)
	if err != nil {
		response.BadRequest(w, r, "invalid delegatorId")
		return
	}
	delegateID, err := uuid.Parse(body.DelegateID)
	if err != nil {
		response.BadRequest(w, r, "invalid delegateId")
		return
	}

	scopeType := body.ScopeType
	if scopeType == "" {
		scopeType = "role"
	}

	var startsAt time.Time
	if body.StartsAt != nil {
		startsAt, err = time.Parse(time.RFC3339, *body.StartsAt)
		if err != nil {
			response.BadRequest(w, r, "invalid startsAt")
			return
		}
	}

	var expiresAt *time.Time
	if body.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *body.ExpiresAt)
		if err != nil {
			response.BadRequest(w, r, "invalid expiresAt")
			return
		}
		expiresAt = &t
	}

	d, err := h.svc.Create(r.Context(), tenantID, delegatorID, delegateID, scopeType, body.GrantedRoles, body.Reason, startsAt, expiresAt)
	if err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}
	response.JSON(w, r, http.StatusCreated, d)
}

func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		response.Unauthorized(w, r)
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequest(w, r, "invalid X-User-ID")
		return
	}

	direction := r.URL.Query().Get("direction")
	var delegations []*Delegation
	if direction == "delegate" {
		delegations, err = h.svc.ListByDelegate(r.Context(), tenantID, userID)
	} else {
		delegations, err = h.svc.ListByDelegator(r.Context(), tenantID, userID)
	}
	if err != nil {
		response.InternalError(w, r)
		return
	}
	response.Collection(w, r, http.StatusOK, delegations, nil)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "invalid delegation id")
		return
	}

	d, err := h.svc.GetByID(r.Context(), tenantID, id)
	if err != nil {
		response.InternalError(w, r)
		return
	}
	if d == nil {
		response.NotFound(w, r)
		return
	}
	response.JSON(w, r, http.StatusOK, d)
}

func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "invalid delegation id")
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		response.Unauthorized(w, r)
		return
	}
	revokedBy, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequest(w, r, "invalid X-User-ID")
		return
	}

	if err := h.svc.Revoke(r.Context(), tenantID, id, revokedBy); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Effective(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}

	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		response.Unauthorized(w, r)
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequest(w, r, "invalid X-User-ID")
		return
	}

	delegations, err := h.svc.GetEffectiveDelegations(r.Context(), tenantID, userID)
	if err != nil {
		response.InternalError(w, r)
		return
	}
	response.Collection(w, r, http.StatusOK, delegations, nil)
}
