package mfa

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
	r.Get("/methods", h.ListMethods)
	r.Post("/enroll", h.Enroll)
	r.Post("/challenge", h.Challenge)
	r.Post("/verify", h.Verify)
}

func (h *Handler) ListMethods(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(r)
	if !ok {
		response.Unauthorized(w, r)
		return
	}
	methods, err := h.svc.ListMethods(r.Context(), userID)
	if err != nil {
		response.InternalError(w, r)
		return
	}
	response.Collection(w, r, http.StatusOK, methods, nil)
}

func (h *Handler) Enroll(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := middleware.GetTenantID(r.Context())
	if !ok {
		response.BadRequest(w, r, "X-Tenant-ID required")
		return
	}
	userID, ok := h.requireUserID(r)
	if !ok {
		response.Unauthorized(w, r)
		return
	}

	var body struct {
		MethodType string `json:"methodType"`
		Name       string `json:"name"`
	}
	if err := validate.DecodeJSON(r, &body); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}
	if body.MethodType == "" {
		body.MethodType = "totp"
	}
	if body.Name == "" {
		body.Name = "default"
	}

	result, method, err := h.svc.Enroll(r.Context(), userID, tenantID, body.MethodType, body.Name)
	if err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}

	response.JSON(w, r, http.StatusCreated, map[string]any{
		"method": method,
		"enroll": result,
	})
}

func (h *Handler) Challenge(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(r)
	if !ok {
		response.Unauthorized(w, r)
		return
	}

	var body struct {
		MethodType string `json:"methodType"`
	}
	if err := validate.DecodeJSON(r, &body); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}
	if body.MethodType == "" {
		body.MethodType = "totp"
	}

	challengeID, err := h.svc.Challenge(r.Context(), userID, body.MethodType)
	if err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, map[string]string{"challengeId": challengeID})
}

func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(r)
	if !ok {
		response.Unauthorized(w, r)
		return
	}

	var body struct {
		ChallengeID string `json:"challengeId"`
		Code        string `json:"code"`
	}
	if err := validate.DecodeJSON(r, &body); err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}

	errs := validate.FieldErrors{}
	validate.Required("challengeId", body.ChallengeID, errs)
	validate.Required("code", body.Code, errs)
	if len(errs) > 0 {
		response.ValidationFailed(w, r, errs)
		return
	}

	ok2, err := h.svc.Verify(r.Context(), userID, body.ChallengeID, body.Code)
	if err != nil {
		response.BadRequest(w, r, err.Error())
		return
	}

	response.JSON(w, r, http.StatusOK, map[string]bool{"verified": ok2})
}

func (h *Handler) requireUserID(r *http.Request) (uuid.UUID, bool) {
	header := r.Header.Get("X-User-ID")
	if header == "" {
		return uuid.UUID{}, false
	}
	id, err := uuid.Parse(header)
	return id, err == nil
}
