package user

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
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Patch("/{id}", h.Update)
	r.Post("/{id}/suspend", h.Suspend)
	r.Post("/{id}/activate", h.Activate)
}

func (h *Handler) requireTenant(r *http.Request) (uuid.UUID, bool) {
	tid, ok := middleware.GetTenantID(r.Context())
	return tid, ok
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.requireTenant(r)
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID header is required")
		return
	}

	var body struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	}
	if err := validate.DecodeJSON(r, &body); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}

	errs := validate.FieldErrors{}
	validate.Required("email", body.Email, errs)
	validate.Email("email", body.Email, errs)
	validate.Required("password", body.Password, errs)
	if len(errs) > 0 {
		response.ValidationFailed(w, r, errs)
		return
	}

	u, err := h.svc.Create(r.Context(), CreateInput{
		TenantID:  tenantID,
		Email:     body.Email,
		Password:  body.Password,
		FirstName: body.FirstName,
		LastName:  body.LastName,
	})
	if err != nil {
		if err.Error() == "email already registered in this tenant" {
			response.Conflict(w, r, err.Error())
			return
		}
		response.BadRequest(w, r, err.Error())
		return
	}

	// Never return the password hash
	u.PasswordHash = ""
	response.JSON(w, r, http.StatusCreated, u)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.requireTenant(r)
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID header is required")
		return
	}

	q := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")
	cursor := r.URL.Query().Get("cursor")

	users, nextCursor, err := h.svc.List(r.Context(), tenantID, q, status, 20, cursor)
	if err != nil {
		response.InternalError(w, r)
		return
	}

	var pagination *response.Pagination
	if nextCursor != "" {
		pagination = &response.Pagination{NextCursor: nextCursor, HasMore: true}
	}

	response.Collection(w, r, http.StatusOK, users, pagination)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.requireTenant(r)
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID header is required")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "invalid user id")
		return
	}

	u, err := h.svc.GetByID(r.Context(), tenantID, id)
	if err != nil {
		response.InternalError(w, r)
		return
	}
	if u == nil {
		response.NotFound(w, r)
		return
	}

	u.PasswordHash = ""
	response.JSON(w, r, http.StatusOK, u)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.requireTenant(r)
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID header is required")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "invalid user id")
		return
	}

	var body struct {
		FirstName  *string `json:"firstName"`
		LastName   *string `json:"lastName"`
		Department *string `json:"department"`
		JobTitle   *string `json:"jobTitle"`
		Status     *string `json:"status"`
	}
	if err := validate.DecodeJSON(r, &body); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}

	u, err := h.svc.Update(r.Context(), tenantID, id, UpdateInput{
		FirstName:  body.FirstName,
		LastName:   body.LastName,
		Department: body.Department,
		JobTitle:   body.JobTitle,
		Status:     body.Status,
	})
	if err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}
	if u == nil {
		response.NotFound(w, r)
		return
	}

	u.PasswordHash = ""
	response.JSON(w, r, http.StatusOK, u)
}

func (h *Handler) Suspend(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, "suspended")
}

func (h *Handler) Activate(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, "active")
}

func (h *Handler) setStatus(w http.ResponseWriter, r *http.Request, status string) {
	tenantID, ok := h.requireTenant(r)
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID header is required")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "invalid user id")
		return
	}

	if err := h.svc.SetStatus(r.Context(), tenantID, id, status); err != nil {
		if err.Error() == "user not found" {
			response.NotFound(w, r)
			return
		}
		response.InternalError(w, r)
		return
	}

	response.JSON(w, r, http.StatusOK, map[string]string{"status": status})
}
