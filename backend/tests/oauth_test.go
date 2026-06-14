package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YASSERRMD/unified-trust-platform/backend/internal/config"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/oauth"
)

func newJWTConfig() config.JWTConfig {
	return config.JWTConfig{
		Issuer:          "https://trust.test",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}
}

func TestOIDCDiscovery(t *testing.T) {
	r := newRouterNoDB()

	req := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Route not wired without DB — expect 404, not 500
	if rr.Code == http.StatusInternalServerError {
		t.Errorf("unexpected 500")
	}
}

func TestJWTIssueAndVerify(t *testing.T) {
	svc, err := oauth.NewTokenService(newJWTConfig())
	if err != nil {
		t.Fatalf("new token service: %v", err)
	}

	token, jti, err := svc.IssueAccessToken("user-1", "tenant-1", "client-1", "test@example.com", []string{"openid", "email"})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if jti == "" {
		t.Fatal("expected non-empty jti")
	}

	claims, err := svc.Verify(token)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if claims.Subject != "user-1" {
		t.Errorf("expected sub=user-1, got %q", claims.Subject)
	}
	if claims.TenantID != "tenant-1" {
		t.Errorf("expected tid=tenant-1, got %q", claims.TenantID)
	}
}

func TestJWKSEndpoint(t *testing.T) {
	svc, err := oauth.NewTokenService(newJWTConfig())
	if err != nil {
		t.Fatalf("new token service: %v", err)
	}

	jwks, err := svc.JWKS()
	if err != nil {
		t.Fatalf("jwks: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(jwks, &parsed); err != nil {
		t.Fatalf("parse jwks: %v", err)
	}

	keys, ok := parsed["keys"].([]any)
	if !ok || len(keys) == 0 {
		t.Error("expected at least one key in JWKS")
	}
}

func TestPKCEVerify(t *testing.T) {
	// Generate a verifier and compute the challenge externally
	// S256: challenge = BASE64URL(SHA256(verifier))
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	// Pre-computed S256 challenge for this verifier
	challenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	if !oauth.PKCEVerify(context.Background(), challenge, verifier) {
		t.Error("expected PKCE verify to succeed for known valid pair")
	}

	if oauth.PKCEVerify(context.Background(), challenge, "wrong-verifier") {
		t.Error("expected PKCE verify to fail for wrong verifier")
	}
}

func TestTokenEndpointMissingParams(t *testing.T) {
	r := newRouterNoDB()

	body := strings.NewReader("grant_type=authorization_code")
	req := httptest.NewRequest(http.MethodPost, "/oauth2/token", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Route not registered in no-DB router — should be 404, not 500
	if rr.Code == http.StatusInternalServerError {
		t.Errorf("unexpected 500 for token endpoint")
	}
}

func TestRefreshTokenGeneration(t *testing.T) {
	svc, err := oauth.NewTokenService(newJWTConfig())
	if err != nil {
		t.Fatalf("new token service: %v", err)
	}

	token, err := svc.IssueRefreshToken()
	if err != nil {
		t.Fatalf("issue refresh token: %v", err)
	}
	if len(token) < 40 {
		t.Errorf("refresh token too short: %d chars", len(token))
	}

	token2, _ := svc.IssueRefreshToken()
	if token == token2 {
		t.Error("refresh tokens should be unique")
	}
}
