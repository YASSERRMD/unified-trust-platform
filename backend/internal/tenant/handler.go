package tenant

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

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
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := validate.DecodeJSON(r, &body); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}

	errs := validate.FieldErrors{}
	validate.Required("name", body.Name, errs)
	validate.Required("slug", body.Slug, errs)
	if len(errs) > 0 {
		response.ValidationFailed(w, r, errs)
		return
	}

	t, err := h.svc.Create(r.Context(), CreateInput{Name: body.Name, Slug: body.Slug})
	if err != nil {
		if err.Error() == "tenant slug already exists" {
			response.Conflict(w, r, err.Error())
			return
		}
		response.BadRequest(w, r, err.Error())
		return
	}

	response.JSON(w, r, http.StatusCreated, t)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	cursor := r.URL.Query().Get("cursor")
	tenants, nextCursor, err := h.svc.List(r.Context(), 20, cursor)
	if err != nil {
		response.InternalError(w, r)
		return
	}

	var pagination *response.Pagination
	if nextCursor != "" {
		pagination = &response.Pagination{NextCursor: nextCursor, HasMore: true}
	}

	response.Collection(w, r, http.StatusOK, tenants, pagination)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "invalid tenant id")
		return
	}

	t, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.InternalError(w, r)
		return
	}
	if t == nil {
		response.NotFound(w, r)
		return
	}

	response.JSON(w, r, http.StatusOK, t)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "invalid tenant id")
		return
	}

	var body struct {
		Name   *string `json:"name"`
		Status *string `json:"status"`
	}
	if err := validate.DecodeJSON(r, &body); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}

	t, err := h.svc.Update(r.Context(), id, UpdateInput{Name: body.Name, Status: body.Status})
	if err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}
	if t == nil {
		response.NotFound(w, r)
		return
	}

	response.JSON(w, r, http.StatusOK, t)
}
