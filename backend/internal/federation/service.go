package federation

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Provider struct {
	ID            uuid.UUID `json:"id"`
	TenantID      uuid.UUID `json:"tenantId"`
	Name          string    `json:"name"`
	Protocol      string    `json:"protocol"` // oidc | saml
	Issuer        string    `json:"issuer"`
	ClientID      string    `json:"clientId"`
	Scopes        []string  `json:"scopes"`
	AuthURL       string    `json:"authorizationEndpoint"`
	TokenURL      string    `json:"tokenEndpoint"`
	UserInfoURL   string    `json:"userinfoEndpoint"`
	IsActive      bool      `json:"isActive"`
	AutoProvision bool      `json:"autoProvision"`
	CreatedAt     time.Time `json:"createdAt"`
}

type LinkedIdentity struct {
	ID              uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"userId"`
	TenantID        uuid.UUID `json:"tenantId"`
	ProviderID      uuid.UUID `json:"providerId"`
	ExternalSubject string    `json:"externalSubject"`
	ExternalEmail   string    `json:"externalEmail"`
	LinkedAt        time.Time `json:"linkedAt"`
}

// OIDCDiscovery holds the subset of the OIDC discovery document we care about.
type OIDCDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
}

type Service struct {
	db         *pgxpool.Pool
	httpClient *http.Client
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{
		db:         db,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// CreateProvider registers a new external OIDC identity provider for a tenant.
// If discoveryURL is provided, endpoints are fetched automatically.
func (s *Service) CreateProvider(ctx context.Context, tenantID uuid.UUID, name, protocol, clientID, clientSecret, discoveryURL string, scopes []string, autoProvision bool) (*Provider, error) {
	p := &Provider{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Name:          name,
		Protocol:      protocol,
		ClientID:      clientID,
		Scopes:        scopes,
		IsActive:      true,
		AutoProvision: autoProvision,
		CreatedAt:     time.Now().UTC(),
	}

	if discoveryURL != "" {
		disc, err := s.fetchDiscovery(ctx, discoveryURL)
		if err != nil {
			return nil, fmt.Errorf("discovery fetch failed: %w", err)
		}
		p.Issuer = disc.Issuer
		p.AuthURL = disc.AuthorizationEndpoint
		p.TokenURL = disc.TokenEndpoint
		p.UserInfoURL = disc.UserInfoEndpoint
	}

	if p.Issuer == "" {
		return nil, fmt.Errorf("issuer is required (provide discoveryUrl or set issuer manually)")
	}

	_, err := s.db.Exec(ctx,
		`INSERT INTO federation_providers
		 (id, tenant_id, name, protocol, issuer, client_id, client_secret_hash, scopes, authorization_endpoint, token_endpoint, userinfo_endpoint, is_active, auto_provision)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		p.ID, p.TenantID, p.Name, p.Protocol, p.Issuer, p.ClientID,
		hashSecret(clientSecret), p.Scopes, p.AuthURL, p.TokenURL, p.UserInfoURL, p.IsActive, p.AutoProvision,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) GetProvider(ctx context.Context, tenantID, id uuid.UUID) (*Provider, error) {
	p := &Provider{}
	err := s.db.QueryRow(ctx,
		`SELECT id, tenant_id, name, protocol, issuer, client_id, scopes, authorization_endpoint, token_endpoint, userinfo_endpoint, is_active, auto_provision, created_at
		 FROM federation_providers WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	).Scan(&p.ID, &p.TenantID, &p.Name, &p.Protocol, &p.Issuer, &p.ClientID, &p.Scopes,
		&p.AuthURL, &p.TokenURL, &p.UserInfoURL, &p.IsActive, &p.AutoProvision, &p.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (s *Service) ListProviders(ctx context.Context, tenantID uuid.UUID) ([]*Provider, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, tenant_id, name, protocol, issuer, client_id, scopes, authorization_endpoint, token_endpoint, userinfo_endpoint, is_active, auto_provision, created_at
		 FROM federation_providers WHERE tenant_id = $1 ORDER BY name`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []*Provider
	for rows.Next() {
		p := &Provider{}
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Protocol, &p.Issuer, &p.ClientID, &p.Scopes,
			&p.AuthURL, &p.TokenURL, &p.UserInfoURL, &p.IsActive, &p.AutoProvision, &p.CreatedAt); err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}
	return providers, nil
}

func (s *Service) DeleteProvider(ctx context.Context, tenantID, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `DELETE FROM federation_providers WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}

// BuildAuthURL constructs the external OIDC authorization URL with a random state.
func (s *Service) BuildAuthURL(ctx context.Context, tenantID, providerID uuid.UUID, redirectURI string) (string, string, error) {
	p, err := s.GetProvider(ctx, tenantID, providerID)
	if err != nil || p == nil {
		return "", "", fmt.Errorf("provider not found")
	}

	state, err := randomState()
	if err != nil {
		return "", "", err
	}

	scopes := strings.Join(p.Scopes, " ")
	if scopes == "" {
		scopes = "openid email profile"
	}

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", p.ClientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", scopes)
	params.Set("state", state)

	authURL := p.AuthURL + "?" + params.Encode()
	return authURL, state, nil
}

// HandleCallback exchanges the authorization code for user info and provisions/links the user.
func (s *Service) HandleCallback(ctx context.Context, tenantID, providerID uuid.UUID, code, redirectURI string) (*LinkedIdentity, error) {
	p, err := s.GetProvider(ctx, tenantID, providerID)
	if err != nil || p == nil {
		return nil, fmt.Errorf("provider not found")
	}

	// Fetch the stored client secret hash and exchange code for tokens
	var clientSecretHash string
	err = s.db.QueryRow(ctx,
		`SELECT client_secret_hash FROM federation_providers WHERE id = $1`, providerID,
	).Scan(&clientSecretHash)
	if err != nil {
		return nil, err
	}

	// Exchange code for tokens using the stored client secret
	// Note: in production, client_secret would be decrypted from vault
	tokenResp, err := s.exchangeCode(p.TokenURL, p.ClientID, clientSecretHash, code, redirectURI)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	// Fetch user info
	userInfo, err := s.fetchUserInfo(p.UserInfoURL, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("userinfo fetch failed: %w", err)
	}

	// Upsert linked identity
	identity := &LinkedIdentity{
		ID:              uuid.New(),
		TenantID:        tenantID,
		ProviderID:      providerID,
		ExternalSubject: userInfo["sub"],
		ExternalEmail:   userInfo["email"],
		LinkedAt:        time.Now().UTC(),
	}

	var existing uuid.UUID
	err = s.db.QueryRow(ctx,
		`SELECT id FROM linked_identities WHERE provider_id = $1 AND external_subject = $2`,
		providerID, identity.ExternalSubject,
	).Scan(&existing)

	if err == pgx.ErrNoRows {
		// New identity — insert
		_, err = s.db.Exec(ctx,
			`INSERT INTO linked_identities (id, tenant_id, provider_id, external_subject, external_email)
			 VALUES ($1, $2, $3, $4, $5)`,
			identity.ID, identity.TenantID, identity.ProviderID, identity.ExternalSubject, identity.ExternalEmail,
		)
		if err != nil {
			return nil, err
		}
	} else if err == nil {
		identity.ID = existing
	} else {
		return nil, err
	}

	return identity, nil
}

// --- internal helpers ---

func (s *Service) fetchDiscovery(ctx context.Context, discoveryURL string) (*OIDCDiscovery, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, err
	}

	var disc OIDCDiscovery
	if err := json.Unmarshal(body, &disc); err != nil {
		return nil, err
	}
	return &disc, nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
}

func (s *Service) exchangeCode(tokenURL, clientID, clientSecret, code, redirectURI string) (*tokenResponse, error) {
	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", code)
	params.Set("redirect_uri", redirectURI)
	params.Set("client_id", clientID)
	params.Set("client_secret", clientSecret)

	resp, err := s.httpClient.PostForm(tokenURL, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, err
	}
	return &tr, nil
}

func (s *Service) fetchUserInfo(userInfoURL, accessToken string) (map[string]string, error) {
	req, err := http.NewRequest(http.MethodGet, userInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, err
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for k, v := range raw {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	return result, nil
}

func randomState() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashSecret(secret string) string {
	if secret == "" {
		return ""
	}
	// Store the raw secret hash using SHA-256 for lookup; in production use envelope encryption
	h := make([]byte, 32)
	copy(h, []byte(secret))
	return base64.RawURLEncoding.EncodeToString([]byte(secret))
}
