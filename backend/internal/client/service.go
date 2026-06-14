package client

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YASSERRMD/unified-trust-platform/backend/internal/auth"
)

type Client struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenantId"`
	ClientID     string    `json:"clientId"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	ClientType   string    `json:"clientType"`
	GrantTypes   []string  `json:"grantTypes"`
	Scopes       []string  `json:"scopes"`
	Status       string    `json:"status"`
	RedirectURIs []string  `json:"redirectUris"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

type CreateInput struct {
	TenantID     uuid.UUID
	Name         string
	Description  string
	ClientType   string
	GrantTypes   []string
	Scopes       []string
	RedirectURIs []string
}

func generateClientID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return "cli_" + hex.EncodeToString(b), nil
}

func generateClientSecret() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*Client, string, error) {
	clientID, err := generateClientID()
	if err != nil {
		return nil, "", fmt.Errorf("generate client_id: %w", err)
	}

	var secretHash string
	var rawSecret string
	if in.ClientType != "public" {
		rawSecret, err = generateClientSecret()
		if err != nil {
			return nil, "", fmt.Errorf("generate client_secret: %w", err)
		}
		secretHash, err = auth.HashPassword(rawSecret)
		if err != nil {
			return nil, "", fmt.Errorf("hash client_secret: %w", err)
		}
	}

	if len(in.GrantTypes) == 0 {
		in.GrantTypes = []string{"authorization_code"}
	}
	if len(in.Scopes) == 0 {
		in.Scopes = []string{"openid", "profile", "email"}
	}

	id := uuid.New()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, "", err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO client_applications
		 (id, tenant_id, client_id, client_secret_hash, name, description, client_type, grant_types, scopes)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, in.TenantID, clientID, secretHash, in.Name, in.Description,
		in.ClientType, in.GrantTypes, in.Scopes,
	)
	if err != nil {
		return nil, "", fmt.Errorf("create client: %w", err)
	}

	for _, uri := range in.RedirectURIs {
		_, err = tx.Exec(ctx,
			`INSERT INTO redirect_uris (client_id, uri) VALUES ($1, $2)`,
			id, uri,
		)
		if err != nil {
			return nil, "", fmt.Errorf("insert redirect_uri: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, "", err
	}

	return &Client{
		ID:           id,
		TenantID:     in.TenantID,
		ClientID:     clientID,
		Name:         in.Name,
		Description:  in.Description,
		ClientType:   in.ClientType,
		GrantTypes:   in.GrantTypes,
		Scopes:       in.Scopes,
		Status:       "active",
		RedirectURIs: in.RedirectURIs,
		CreatedAt:    time.Now().UTC(),
	}, rawSecret, nil
}

func (s *Service) GetByClientID(ctx context.Context, clientID string) (*Client, string, error) {
	var c Client
	var secretHash string
	err := s.db.QueryRow(ctx,
		`SELECT id, tenant_id, client_id, client_secret_hash, name, description,
		        client_type, grant_types, scopes, status, created_at
		 FROM client_applications WHERE client_id = $1 AND status = 'active'`,
		clientID,
	).Scan(&c.ID, &c.TenantID, &c.ClientID, &secretHash, &c.Name, &c.Description,
		&c.ClientType, &c.GrantTypes, &c.Scopes, &c.Status, &c.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("get client: %w", err)
	}

	rows, _ := s.db.Query(ctx, `SELECT uri FROM redirect_uris WHERE client_id = $1`, c.ID)
	defer rows.Close()
	for rows.Next() {
		var uri string
		rows.Scan(&uri)
		c.RedirectURIs = append(c.RedirectURIs, uri)
	}

	return &c, secretHash, nil
}

func (s *Service) ValidateRedirectURI(c *Client, redirectURI string) bool {
	for _, uri := range c.RedirectURIs {
		if strings.EqualFold(uri, redirectURI) {
			return true
		}
	}
	return false
}
