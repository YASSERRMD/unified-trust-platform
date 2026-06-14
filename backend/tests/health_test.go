package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/handler"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/router"
)

func TestHealthEndpoint(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := handler.NewHealthHandler(nil, nil)
	r := router.New(logger, h)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data field, got %v", resp)
	}

	if data["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", data["status"])
	}
}

func TestMetaEndpoint(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := handler.NewHealthHandler(nil, nil)
	r := router.New(logger, h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data field")
	}

	if data["version"] == nil {
		t.Error("expected version field")
	}
}

func TestNotFoundReturnsStandardError(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := handler.NewHealthHandler(nil, nil)
	r := router.New(logger, h)

	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["error"] == nil {
		t.Error("expected error field in 404 response")
	}
}

func TestRequestIDHeader(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	h := handler.NewHealthHandler(nil, nil)
	r := router.New(logger, h)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header in response")
	}
}
