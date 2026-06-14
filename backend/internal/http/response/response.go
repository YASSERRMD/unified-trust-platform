package response

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/middleware"
)

type Meta struct {
	RequestID string `json:"requestId"`
	Timestamp string `json:"timestamp"`
}

type Envelope struct {
	Data interface{} `json:"data,omitempty"`
	Meta Meta        `json:"meta"`
}

type CollectionEnvelope struct {
	Data       interface{} `json:"data"`
	Pagination *Pagination `json:"pagination,omitempty"`
	Meta       Meta        `json:"meta"`
}

type Pagination struct {
	NextCursor string `json:"nextCursor,omitempty"`
	HasMore    bool   `json:"hasMore"`
	Total      int64  `json:"total,omitempty"`
}

type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type ErrorEnvelope struct {
	Error ErrorDetail `json:"error"`
	Meta  Meta        `json:"meta"`
}

func buildMeta(r *http.Request) Meta {
	return Meta{
		RequestID: middleware.GetRequestID(r.Context()),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func JSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Envelope{Data: data, Meta: buildMeta(r)})
}

func Collection(w http.ResponseWriter, r *http.Request, status int, data interface{}, pagination *Pagination) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(CollectionEnvelope{Data: data, Pagination: pagination, Meta: buildMeta(r)})
}

func Error(w http.ResponseWriter, r *http.Request, status int, code, message string, details interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorEnvelope{
		Error: ErrorDetail{Code: code, Message: message, Details: details},
		Meta:  buildMeta(r),
	})
}

func BadRequest(w http.ResponseWriter, r *http.Request, message string) {
	Error(w, r, http.StatusBadRequest, "INVALID_REQUEST", message, nil)
}

func Unauthorized(w http.ResponseWriter, r *http.Request) {
	Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", nil)
}

func Forbidden(w http.ResponseWriter, r *http.Request) {
	Error(w, r, http.StatusForbidden, "FORBIDDEN", "access denied", nil)
}

func NotFound(w http.ResponseWriter, r *http.Request) {
	Error(w, r, http.StatusNotFound, "NOT_FOUND", "resource not found", nil)
}

func Conflict(w http.ResponseWriter, r *http.Request, message string) {
	Error(w, r, http.StatusConflict, "CONFLICT", message, nil)
}

func ValidationFailed(w http.ResponseWriter, r *http.Request, details interface{}) {
	Error(w, r, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "validation failed", details)
}

func InternalError(w http.ResponseWriter, r *http.Request) {
	Error(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred", nil)
}
