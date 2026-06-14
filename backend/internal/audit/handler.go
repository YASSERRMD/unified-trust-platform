package audit

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/middleware"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/response"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/", h.List)
	r.Get("/export", h.Export)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}

	f := ListFilter{}

	if v := r.URL.Query().Get("actorId"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			response.BadRequest(w, r, "invalid actorId")
			return
		}
		f.ActorID = &id
	}
	f.Action = r.URL.Query().Get("action")
	f.Resource = r.URL.Query().Get("resource")
	f.Outcome = r.URL.Query().Get("outcome")
	f.Cursor = r.URL.Query().Get("cursor")

	if v := r.URL.Query().Get("since"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			response.BadRequest(w, r, "invalid since (RFC3339 required)")
			return
		}
		f.Since = &t
	}
	if v := r.URL.Query().Get("until"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			response.BadRequest(w, r, "invalid until (RFC3339 required)")
			return
		}
		f.Until = &t
	}

	events, nextCursor, err := h.svc.List(r.Context(), tenantID, f)
	if err != nil {
		response.InternalError(w, r)
		return
	}

	var pagination *response.Pagination
	if nextCursor != "" {
		pagination = &response.Pagination{NextCursor: nextCursor, HasMore: true}
	}
	response.Collection(w, r, http.StatusOK, events, pagination)
}

func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}

	f := ListFilter{}
	f.Action = r.URL.Query().Get("action")
	f.Resource = r.URL.Query().Get("resource")
	f.Outcome = r.URL.Query().Get("outcome")

	if v := r.URL.Query().Get("since"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			response.BadRequest(w, r, "invalid since")
			return
		}
		f.Since = &t
	}
	if v := r.URL.Query().Get("until"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			response.BadRequest(w, r, "invalid until")
			return
		}
		f.Until = &t
	}

	csv, err := h.svc.ExportCSV(r.Context(), tenantID, f)
	if err != nil {
		response.InternalError(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\"audit-events.csv\"")
	w.WriteHeader(http.StatusOK)
	w.Write(csv)
}
