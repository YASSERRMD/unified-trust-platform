package jit

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

func (h *Handler) Routes(r chi.Router) {
	r.Post("/", h.CreateRequest)
	r.Get("/", h.ListRequests)
	r.Get("/{id}", h.GetRequest)
	r.Post("/{id}/approve", h.Approve)
	r.Post("/{id}/deny", h.Deny)
}

func (h *Handler) GrantRoutes(r chi.Router) {
	r.Get("/active", h.ActiveGrants)
	r.Delete("/{grantId}", h.RevokeGrant)
}

func (h *Handler) requireUserID(r *http.Request) (uuid.UUID, bool) {
	s := r.Header.Get("X-User-ID")
	if s == "" {
		return uuid.UUID{}, false
	}
	id, err := uuid.Parse(s)
	return id, err == nil
}

func (h *Handler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}
	requesterID, ok := h.requireUserID(r)
	if !ok {
		response.Unauthorized(w, r)
		return
	}

	var body struct {
		ResourceType    string `json:"resourceType"`
		ResourceID      string `json:"resourceId"`
		RequestedRole   string `json:"requestedRole"`
		Justification   string `json:"justification"`
		DurationMinutes int    `json:"requestedDurationMinutes"`
	}
	if err := validate.DecodeJSON(r, &body); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}

	errs := validate.FieldErrors{}
	validate.Required("requestedRole", body.RequestedRole, errs)
	validate.Required("justification", body.Justification, errs)
	if body.DurationMinutes <= 0 {
		errs["requestedDurationMinutes"] = "must be greater than 0"
	}
	if len(errs) > 0 {
		response.ValidationFailed(w, r, errs)
		return
	}

	req, err := h.svc.CreateRequest(r.Context(), tenantID, requesterID, body.ResourceType, body.ResourceID,
		body.RequestedRole, body.Justification, body.DurationMinutes)
	if err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}
	response.JSON(w, r, http.StatusCreated, req)
}

func (h *Handler) ListRequests(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}

	status := r.URL.Query().Get("status")
	reqs, err := h.svc.ListRequests(r.Context(), tenantID, status)
	if err != nil {
		response.InternalError(w, r)
		return
	}
	response.Collection(w, r, http.StatusOK, reqs, nil)
}

func (h *Handler) GetRequest(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "invalid request id")
		return
	}

	req, err := h.svc.GetRequest(r.Context(), tenantID, id)
	if err != nil {
		response.InternalError(w, r)
		return
	}
	if req == nil {
		response.NotFound(w, r)
		return
	}
	response.JSON(w, r, http.StatusOK, req)
}

func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}
	approverID, ok := h.requireUserID(r)
	if !ok {
		response.Unauthorized(w, r)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "invalid request id")
		return
	}

	grant, err := h.svc.Approve(r.Context(), tenantID, id, approverID)
	if err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}
	response.JSON(w, r, http.StatusOK, grant)
}

func (h *Handler) Deny(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "invalid request id")
		return
	}

	if err := h.svc.Deny(r.Context(), tenantID, id); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}
	response.JSON(w, r, http.StatusOK, map[string]string{"status": "denied"})
}

func (h *Handler) ActiveGrants(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}
	requesterID, ok := h.requireUserID(r)
	if !ok {
		response.Unauthorized(w, r)
		return
	}

	grants, err := h.svc.GetActiveGrants(r.Context(), tenantID, requesterID)
	if err != nil {
		response.InternalError(w, r)
		return
	}
	response.Collection(w, r, http.StatusOK, grants, nil)
}

func (h *Handler) RevokeGrant(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}
	grantID, err := uuid.Parse(chi.URLParam(r, "grantId"))
	if err != nil {
		response.BadRequest(w, r, "invalid grantId")
		return
	}

	if err := h.svc.RevokeGrant(r.Context(), tenantID, grantID); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
