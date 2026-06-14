package policy

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/middleware"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/response"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/validate"
)

type PolicyHandler struct {
	engine *PolicyEngine
}

func NewPolicyHandler(engine *PolicyEngine) *PolicyHandler {
	return &PolicyHandler{engine: engine}
}

func (h *PolicyHandler) PolicyRoutes(r chi.Router) {
	r.Post("/", h.CreatePolicy)
	r.Get("/", h.ListPolicies)
	r.Get("/{id}", h.GetPolicy)
	r.Put("/{id}", h.UpdatePolicy)
	r.Delete("/{id}", h.DeletePolicy)
}

func (h *PolicyHandler) EvaluateRoute(r chi.Router) {
	r.Post("/evaluate", h.Evaluate)
}

func (h *PolicyHandler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}

	var body struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Effect      string          `json:"effect"`
		Priority    int             `json:"priority"`
		Subjects    PolicySubjects  `json:"subjects"`
		Actions     []string        `json:"actions"`
		Resources   PolicyResources `json:"resources"`
		Conditions  []Condition     `json:"conditions"`
	}
	if err := validate.DecodeJSON(r, &body); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}

	errs := validate.FieldErrors{}
	validate.Required("name", body.Name, errs)
	validate.OneOf("effect", body.Effect, []string{"permit", "deny"}, errs)
	if len(body.Actions) == 0 {
		errs["actions"] = "at least one action required"
	}
	if len(errs) > 0 {
		response.ValidationFailed(w, r, errs)
		return
	}

	if body.Priority == 0 {
		body.Priority = 100
	}

	p := &PolicyDocument{
		Name:        body.Name,
		Description: body.Description,
		Effect:      body.Effect,
		Priority:    body.Priority,
		Subjects:    body.Subjects,
		Actions:     body.Actions,
		Resources:   body.Resources,
		Conditions:  body.Conditions,
	}

	if err := h.engine.CreatePolicy(r.Context(), tenantID, p); err != nil {
		response.InternalError(w, r)
		return
	}
	response.JSON(w, r, http.StatusCreated, p)
}

func (h *PolicyHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}
	policies, err := h.engine.loadPolicies(r.Context(), tenantID)
	if err != nil {
		response.InternalError(w, r)
		return
	}
	response.Collection(w, r, http.StatusOK, policies, nil)
}

func (h *PolicyHandler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "invalid policy id")
		return
	}
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}

	p, err := h.engine.getPolicyByID(r.Context(), tenantID, id)
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

func (h *PolicyHandler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "invalid policy id")
		return
	}
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}

	var body struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Effect      string          `json:"effect"`
		Priority    int             `json:"priority"`
		Subjects    PolicySubjects  `json:"subjects"`
		Actions     []string        `json:"actions"`
		Resources   PolicyResources `json:"resources"`
		Conditions  []Condition     `json:"conditions"`
		IsActive    *bool           `json:"isActive"`
	}
	if err := validate.DecodeJSON(r, &body); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}

	if err := h.engine.updatePolicy(r.Context(), tenantID, id, body.Name, body.Description, body.Effect,
		body.Priority, body.Subjects, body.Actions, body.Resources, body.Conditions, body.IsActive); err != nil {
		response.InternalError(w, r)
		return
	}
	response.JSON(w, r, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *PolicyHandler) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "invalid policy id")
		return
	}
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}

	if err := h.engine.deletePolicy(r.Context(), tenantID, id); err != nil {
		response.InternalError(w, r)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *PolicyHandler) Evaluate(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}

	var req EvalRequest
	if err := validate.DecodeJSON(r, &req); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}

	errs := validate.FieldErrors{}
	validate.Required("subject.userId", req.Subject.UserID, errs)
	validate.Required("action", req.Action, errs)
	if len(errs) > 0 {
		response.ValidationFailed(w, r, errs)
		return
	}

	if req.Env.Time.IsZero() {
		req.Env.Time = time.Now().UTC()
	}

	decision, err := h.engine.Evaluate(r.Context(), tenantID, req)
	if err != nil {
		response.InternalError(w, r)
		return
	}
	response.JSON(w, r, http.StatusOK, decision)
}
