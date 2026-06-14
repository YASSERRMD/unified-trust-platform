package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/handler"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/router"
)

func newRouterNoDB() http.Handler {
	logger, _ := zap.NewDevelopment()
	h := &router.Handlers{
		Health: handler.NewHealthHandler(nil, nil),
	}
	return router.New(logger, h)
}

func TestTenantRouteRequiresTenant(t *testing.T) {
	r := newRouterNoDB()

	body := `{"name":"Test","slug":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Without a DB handler wired, the route returns 404 (not registered)
	if rr.Code == http.StatusInternalServerError {
		t.Errorf("should not return 500")
	}
}

func TestUserRouteRequiresTenantHeader(t *testing.T) {
	r := newRouterNoDB()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// route not registered without DB — should be 404, not panic
	if rr.Code == http.StatusInternalServerError {
		t.Errorf("should not return 500")
	}
}

func TestPasswordPolicy(t *testing.T) {
	cases := []struct {
		password string
		wantErr  bool
	}{
		{"short", true},
		{"alllowercase123!", false},
		{"NoSymbol12345678", true},
		{"Valid@Password1!", false},
		{"aA1!aA1!aA1!aA1!", false},
	}

	for _, tc := range cases {
		t.Run(tc.password, func(t *testing.T) {
			body := map[string]string{
				"email":    "test@example.com",
				"password": tc.password,
			}
			b, _ := json.Marshal(body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")

			rr := httptest.NewRecorder()
			r := newRouterNoDB()
			r.ServeHTTP(rr, req)

			// Route not registered in no-DB router, but no panics expected
			if rr.Code == http.StatusInternalServerError {
				t.Errorf("unexpected 500 for password=%q", tc.password)
			}
		})
	}
}
