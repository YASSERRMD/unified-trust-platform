package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	clientpkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/client"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/config"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/response"
	userpkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/user"
)

type Handler struct {
	db          *pgxpool.Pool
	tokenSvc    *TokenService
	clientSvc   *clientpkg.Service
	userSvc     *userpkg.Service
	authCodeStr *AuthCodeStore
	cfg         config.JWTConfig
}

func NewHandler(db *pgxpool.Pool, cfg config.JWTConfig, clientSvc *clientpkg.Service, userSvc *userpkg.Service) (*Handler, error) {
	tokenSvc, err := NewTokenService(cfg)
	if err != nil {
		return nil, err
	}
	return &Handler{
		db:          db,
		tokenSvc:    tokenSvc,
		clientSvc:   clientSvc,
		userSvc:     userSvc,
		authCodeStr: NewAuthCodeStore(db),
		cfg:         cfg,
	}, nil
}

// Discovery serves /.well-known/openid-configuration
func (h *Handler) Discovery(w http.ResponseWriter, r *http.Request) {
	issuer := h.cfg.Issuer
	doc := map[string]any{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + "/oauth2/authorize",
		"token_endpoint":                        issuer + "/oauth2/token",
		"userinfo_endpoint":                     issuer + "/oauth2/userinfo",
		"jwks_uri":                              issuer + "/.well-known/jwks.json",
		"revocation_endpoint":                   issuer + "/oauth2/revoke",
		"end_session_endpoint":                  issuer + "/oauth2/logout",
		"scopes_supported":                      []string{"openid", "profile", "email", "offline_access"},
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post", "client_secret_basic"},
		"code_challenge_methods_supported":      []string{"S256"},
		"claims_supported":                      []string{"sub", "iss", "aud", "exp", "iat", "email", "name", "tid"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

// JWKS serves /.well-known/jwks.json
func (h *Handler) JWKS(w http.ResponseWriter, r *http.Request) {
	data, err := h.tokenSvc.JWKS()
	if err != nil {
		response.InternalError(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// Authorize handles GET /oauth2/authorize
func (h *Handler) Authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	responseType := q.Get("response_type")
	scope := q.Get("scope")
	state := q.Get("state")
	codeChallenge := q.Get("code_challenge")
	codeChallengeMethod := q.Get("code_challenge_method")
	nonce := q.Get("nonce")

	if responseType != "code" {
		oauthError(w, r, redirectURI, state, "unsupported_response_type", "only code response_type is supported")
		return
	}

	if codeChallenge == "" || codeChallengeMethod != "S256" {
		oauthError(w, r, redirectURI, state, "invalid_request", "S256 code_challenge required")
		return
	}

	c, _, err := h.clientSvc.GetByClientID(r.Context(), clientID)
	if err != nil || c == nil {
		oauthError(w, r, redirectURI, state, "invalid_client", "unknown client_id")
		return
	}

	if !h.clientSvc.ValidateRedirectURI(c, redirectURI) {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}

	// In production this redirects to the login UI.
	// For the API layer, return JSON describing what is needed.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"status":                "login_required",
		"client_id":             clientID,
		"redirect_uri":          redirectURI,
		"scope":                 scope,
		"state":                 state,
		"code_challenge":        codeChallenge,
		"code_challenge_method": codeChallengeMethod,
		"nonce":                 nonce,
		"message":               "submit credentials to /oauth2/authorize/complete",
	})
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// Token handles POST /oauth2/token
func (h *Handler) Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		tokenError(w, "invalid_request", "failed to parse form")
		return
	}

	switch r.FormValue("grant_type") {
	case "authorization_code":
		h.tokenAuthCode(w, r)
	case "refresh_token":
		tokenError(w, "invalid_grant", "refresh_token grant not yet implemented")
	default:
		tokenError(w, "unsupported_grant_type", fmt.Sprintf("grant_type %q not supported", r.FormValue("grant_type")))
	}
}

func (h *Handler) tokenAuthCode(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	redirectURI := r.FormValue("redirect_uri")
	clientID := r.FormValue("client_id")
	codeVerifier := r.FormValue("code_verifier")

	if code == "" || redirectURI == "" || clientID == "" || codeVerifier == "" {
		tokenError(w, "invalid_request", "code, redirect_uri, client_id, code_verifier are required")
		return
	}

	c, secretHash, err := h.clientSvc.GetByClientID(r.Context(), clientID)
	if err != nil || c == nil {
		tokenError(w, "invalid_client", "unknown client")
		return
	}

	if c.ClientType == "confidential" {
		secret := r.FormValue("client_secret")
		if secret == "" {
			_, pass, ok := r.BasicAuth()
			if ok {
				secret = pass
			}
		}
		if secretHash != "" && !checkSecret(secretHash, secret) {
			tokenError(w, "invalid_client", "invalid client_secret")
			return
		}
	}

	ac, err := h.authCodeStr.Consume(r.Context(), code)
	if err != nil || ac == nil {
		tokenError(w, "invalid_grant", "code is invalid or expired")
		return
	}

	if ac.RedirectURI != redirectURI {
		tokenError(w, "invalid_grant", "redirect_uri mismatch")
		return
	}

	if !PKCEVerify(r.Context(), ac.CodeChallenge, codeVerifier) {
		tokenError(w, "invalid_grant", "code_verifier mismatch")
		return
	}

	u, err := h.userSvc.GetByID(r.Context(), ac.TenantID, ac.UserID)
	if err != nil || u == nil {
		tokenError(w, "server_error", "user not found")
		return
	}

	accessToken, _, err := h.tokenSvc.IssueAccessToken(
		u.ID.String(), u.TenantID.String(), clientID, u.Email, ac.Scopes,
	)
	if err != nil {
		tokenError(w, "server_error", "failed to issue access token")
		return
	}

	idToken, err := h.tokenSvc.IssueIDToken(
		u.ID.String(), u.TenantID.String(), clientID, ac.Nonce, u.Email, ac.Scopes,
	)
	if err != nil {
		tokenError(w, "server_error", "failed to issue id token")
		return
	}

	refreshToken, err := h.tokenSvc.IssueRefreshToken()
	if err != nil {
		tokenError(w, "server_error", "failed to issue refresh token")
		return
	}

	_ = h.storeRefreshToken(r.Context(), u.ID, u.TenantID, c.ID, ac.Scopes, refreshToken)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(h.cfg.AccessTokenTTL.Seconds()),
		RefreshToken: refreshToken,
		IDToken:      idToken,
		Scope:        ScopeString(ac.Scopes),
	})
}

// Revoke handles POST /oauth2/revoke — RFC 7009 always returns 200.
func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// UserInfo handles GET /oauth2/userinfo
func (h *Handler) UserInfo(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		response.Unauthorized(w, r)
		return
	}

	claims, err := h.tokenSvc.Verify(strings.TrimPrefix(authHeader, "Bearer "))
	if err != nil {
		response.Unauthorized(w, r)
		return
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		response.Unauthorized(w, r)
		return
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		response.Unauthorized(w, r)
		return
	}

	u, err := h.userSvc.GetByID(r.Context(), tenantID, userID)
	if err != nil || u == nil {
		response.Unauthorized(w, r)
		return
	}

	info := map[string]any{
		"sub": u.ID.String(),
		"tid": u.TenantID.String(),
	}
	for _, scope := range claims.Scopes {
		if scope == "email" {
			info["email"] = u.Email
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// Logout handles POST /oauth2/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if redirect := r.URL.Query().Get("post_logout_redirect_uri"); redirect != "" {
		http.Redirect(w, r, redirect, http.StatusFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) storeRefreshToken(ctx context.Context, userID, tenantID, clientID uuid.UUID, scopes []string, rawToken string) error {
	hash := sha256Hex(rawToken)
	_, err := h.db.Exec(ctx,
		`INSERT INTO tokens (user_id, tenant_id, client_id, token_type, token_hash, scopes, expires_at)
		 VALUES ($1, $2, $3, 'refresh', $4, $5, $6)`,
		userID, tenantID, clientID, hash, scopes, time.Now().UTC().Add(7*24*time.Hour),
	)
	return err
}

func oauthError(w http.ResponseWriter, r *http.Request, redirectURI, state, errCode, desc string) {
	if redirectURI != "" {
		u, err := url.Parse(redirectURI)
		if err == nil {
			q := u.Query()
			q.Set("error", errCode)
			q.Set("error_description", desc)
			if state != "" {
				q.Set("state", state)
			}
			u.RawQuery = q.Encode()
			http.Redirect(w, r, u.String(), http.StatusFound)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             errCode,
		"error_description": desc,
	})
}

func tokenError(w http.ResponseWriter, errCode, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             errCode,
		"error_description": desc,
	})
}
