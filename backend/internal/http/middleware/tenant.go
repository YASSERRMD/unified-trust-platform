package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

const tenantIDKey contextKey = "tenantID"

func TenantContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantHeader := r.Header.Get("X-Tenant-ID")
		if tenantHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		tenantID, err := uuid.Parse(tenantHeader)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "INVALID_REQUEST",
					"message": "invalid X-Tenant-ID header",
				},
			})
			return
		}

		ctx := context.WithValue(r.Context(), tenantIDKey, tenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetTenantID(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(tenantIDKey).(uuid.UUID)
	return v, ok
}
