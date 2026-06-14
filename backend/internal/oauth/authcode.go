package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthCode struct {
	ID                  uuid.UUID
	CodeHash            string
	ClientID            uuid.UUID
	UserID              uuid.UUID
	TenantID            uuid.UUID
	RedirectURI         string
	Scopes              []string
	CodeChallenge       string
	CodeChallengeMethod string
	Nonce               string
	ExpiresAt           time.Time
}

type AuthCodeStore struct {
	db *pgxpool.Pool
}

func NewAuthCodeStore(db *pgxpool.Pool) *AuthCodeStore {
	return &AuthCodeStore{db: db}
}

func (s *AuthCodeStore) Create(ctx context.Context, clientID, userID, tenantID uuid.UUID, redirectURI string, scopes []string, challenge, method, nonce string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	code := hex.EncodeToString(raw)
	h := sha256.Sum256([]byte(code))
	codeHash := hex.EncodeToString(h[:])

	_, err := s.db.Exec(ctx,
		`INSERT INTO authorization_codes
		 (code_hash, client_id, user_id, tenant_id, redirect_uri, scopes, code_challenge, code_challenge_method, nonce, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		codeHash, clientID, userID, tenantID, redirectURI, scopes,
		challenge, method, nonce, time.Now().UTC().Add(5*time.Minute),
	)
	if err != nil {
		return "", fmt.Errorf("store auth code: %w", err)
	}

	return code, nil
}

func (s *AuthCodeStore) Consume(ctx context.Context, code string) (*AuthCode, error) {
	h := sha256.Sum256([]byte(code))
	codeHash := hex.EncodeToString(h[:])

	var ac AuthCode
	err := s.db.QueryRow(ctx,
		`SELECT id, code_hash, client_id, user_id, tenant_id, redirect_uri, scopes,
		        code_challenge, code_challenge_method, nonce, expires_at
		 FROM authorization_codes
		 WHERE code_hash = $1 AND is_used = FALSE AND expires_at > NOW()`,
		codeHash,
	).Scan(
		&ac.ID, &ac.CodeHash, &ac.ClientID, &ac.UserID, &ac.TenantID,
		&ac.RedirectURI, &ac.Scopes, &ac.CodeChallenge, &ac.CodeChallengeMethod,
		&ac.Nonce, &ac.ExpiresAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("consume auth code: %w", err)
	}

	_, err = s.db.Exec(ctx, `UPDATE authorization_codes SET is_used = TRUE WHERE id = $1`, ac.ID)
	if err != nil {
		return nil, fmt.Errorf("mark code used: %w", err)
	}

	return &ac, nil
}

// ScopeString returns scopes as a space-separated string.
func ScopeString(scopes []string) string {
	return strings.Join(scopes, " ")
}

// ScopeSlice splits a scope string into a slice.
func ScopeSlice(scope string) []string {
	return strings.Fields(scope)
}
