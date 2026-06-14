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

type RBACHandler struct {
	svc *RBACService
}

func NewRBACHandler(svc *RBACService) *RBACHandler {
	return &RBACHandler{svc: svc}
}

func (h *RBACHandler) RoleRoutes(r chi.Router) {
	r.Post("/", h.CreateRole)
	r.Get("/", h.ListRoles)
	r.Get("/{id}", h.GetRole)
	r.Post("/{id}/permissions", h.AssignPermission)
}

func (h *RBACHandler) PermissionRoutes(r chi.Router) {
	r.Post("/", h.CreatePermission)
	r.Get("/", h.ListPermissions)
}

func (h *RBACHandler) UserRoleRoutes(r chi.Router) {
	r.Post("/", h.AssignUserRole)
	r.Delete("/{roleId}", h.RemoveUserRole)
}

func (h *RBACHandler) EvaluateRoute(r chi.Router) {
	r.Post("/evaluate", h.Evaluate)
}

func (h *RBACHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := validate.DecodeJSON(r, &body); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}
	errs := validate.FieldErrors{}
	validate.Required("name", body.Name, errs)
	if len(errs) > 0 {
		response.ValidationFailed(w, r, errs)
		return
	}
	role, err := h.svc.CreateRole(r.Context(), tenantID, body.Name, body.Description)
	if err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}
	response.JSON(w, r, http.StatusCreated, role)
}

func (h *RBACHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}
	roles, err := h.svc.ListRoles(r.Context(), tenantID)
	if err != nil {
		response.InternalError(w, r)
		return
	}
	response.Collection(w, r, http.StatusOK, roles, nil)
}

func (h *RBACHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "invalid role id")
		return
	}
	role, err := h.svc.GetRole(r.Context(), id)
	if err != nil {
		response.InternalError(w, r)
		return
	}
	if role == nil {
		response.NotFound(w, r)
		return
	}
	response.JSON(w, r, http.StatusOK, role)
}

func (h *RBACHandler) AssignPermission(w http.ResponseWriter, r *http.Request) {
	roleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, r, "invalid role id")
		return
	}
	var body struct {
		PermissionID string `json:"permissionId"`
	}
	if err := validate.DecodeJSON(r, &body); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}
	permID, err := uuid.Parse(body.PermissionID)
	if err != nil {
		response.BadRequest(w, r, "invalid permissionId")
		return
	}
	if err := h.svc.AssignRolePermission(r.Context(), roleID, permID); err != nil {
		response.InternalError(w, r)
		return
	}
	response.JSON(w, r, http.StatusOK, map[string]string{"status": "assigned"})
}

func (h *RBACHandler) CreatePermission(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}
	var body struct {
		Name        string `json:"name"`
		Resource    string `json:"resource"`
		Action      string `json:"action"`
		Description string `json:"description"`
	}
	if err := validate.DecodeJSON(r, &body); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}
	errs := validate.FieldErrors{}
	validate.Required("resource", body.Resource, errs)
	validate.Required("action", body.Action, errs)
	if len(errs) > 0 {
		response.ValidationFailed(w, r, errs)
		return
	}
	if body.Name == "" {
		body.Name = body.Resource + ":" + body.Action
	}
	perm, err := h.svc.CreatePermission(r.Context(), tenantID, body.Name, body.Resource, body.Action, body.Description)
	if err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}
	response.JSON(w, r, http.StatusCreated, perm)
}

func (h *RBACHandler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}
	perms, err := h.svc.ListPermissions(r.Context(), tenantID)
	if err != nil {
		response.InternalError(w, r)
		return
	}
	response.Collection(w, r, http.StatusOK, perms, nil)
}

func (h *RBACHandler) AssignUserRole(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		response.BadRequest(w, r, "invalid userId")
		return
	}
	var body struct {
		RoleID    string  `json:"roleId"`
		ExpiresAt *string `json:"expiresAt"`
	}
	if err := validate.DecodeJSON(r, &body); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}
	roleID, err := uuid.Parse(body.RoleID)
	if err != nil {
		response.BadRequest(w, r, "invalid roleId")
		return
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
	if err := h.svc.AssignUserRole(r.Context(), userID, roleID, tenantID, nil, expiresAt); err != nil {
		response.InternalError(w, r)
		return
	}
	response.JSON(w, r, http.StatusOK, map[string]string{"status": "assigned"})
}

func (h *RBACHandler) RemoveUserRole(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		response.BadRequest(w, r, "invalid userId")
		return
	}
	roleID, err := uuid.Parse(chi.URLParam(r, "roleId"))
	if err != nil {
		response.BadRequest(w, r, "invalid roleId")
		return
	}
	if err := h.svc.RemoveUserRole(r.Context(), userID, roleID); err != nil {
		response.InternalError(w, r)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RBACHandler) Evaluate(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}
	var body struct {
		UserID   string `json:"userId"`
		Resource string `json:"resource"`
		Action   string `json:"action"`
	}
	if err := validate.DecodeJSON(r, &body); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}
	uid, err := uuid.Parse(body.UserID)
	if err != nil {
		response.BadRequest(w, r, "invalid userId")
		return
	}
	allowed, err := h.svc.HasPermission(r.Context(), uid, tenantID, body.Resource, body.Action)
	if err != nil {
		response.InternalError(w, r)
		return
	}
	decision := "deny"
	if allowed {
		decision = "permit"
	}
	response.JSON(w, r, http.StatusOK, map[string]any{
		"allowed":  allowed,
		"decision": decision,
		"reason":   "RBAC evaluation",
	})
}
