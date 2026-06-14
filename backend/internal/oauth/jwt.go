package oauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/YASSERRMD/unified-trust-platform/backend/internal/config"
)

type TokenService struct {
	privateKey *rsa.PrivateKey
	keyID      string
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewTokenService(cfg config.JWTConfig) (*TokenService, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate RSA key: %w", err)
	}

	return &TokenService{
		privateKey: key,
		keyID:      uuid.New().String(),
		issuer:     cfg.Issuer,
		accessTTL:  cfg.AccessTokenTTL,
		refreshTTL: cfg.RefreshTokenTTL,
	}, nil
}

type Claims struct {
	jwt.RegisteredClaims
	TenantID string   `json:"tid"`
	Email    string   `json:"email,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
	TokenUse string   `json:"token_use"`
}

func (ts *TokenService) IssueAccessToken(userID, tenantID, clientID, email string, scopes []string) (string, string, error) {
	jti := uuid.New().String()
	now := time.Now().UTC()

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ts.issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{clientID},
			ExpiresAt: jwt.NewNumericDate(now.Add(ts.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        jti,
		},
		TenantID: tenantID,
		Email:    email,
		Scopes:   scopes,
		TokenUse: "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = ts.keyID

	signed, err := token.SignedString(ts.privateKey)
	if err != nil {
		return "", "", fmt.Errorf("sign access token: %w", err)
	}

	return signed, jti, nil
}

func (ts *TokenService) IssueIDToken(userID, tenantID, clientID, nonce, email string, scopes []string) (string, error) {
	now := time.Now().UTC()

	mapClaims := jwt.MapClaims{
		"iss":       ts.issuer,
		"sub":       userID,
		"aud":       clientID,
		"exp":       now.Add(ts.accessTTL).Unix(),
		"iat":       now.Unix(),
		"jti":       uuid.New().String(),
		"tid":       tenantID,
		"token_use": "id",
	}

	for _, scope := range scopes {
		if scope == "email" {
			mapClaims["email"] = email
		}
		if scope == "profile" {
			mapClaims["name"] = email
		}
	}

	if nonce != "" {
		mapClaims["nonce"] = nonce
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, mapClaims)
	token.Header["kid"] = ts.keyID

	return token.SignedString(ts.privateKey)
}

func (ts *TokenService) IssueRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (ts *TokenService) Verify(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return &ts.privateKey.PublicKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

func (ts *TokenService) JWKS() ([]byte, error) {
	pub := &ts.privateKey.PublicKey

	jwk := map[string]any{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": ts.keyID,
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}

	return json.Marshal(map[string]any{
		"keys": []any{jwk},
	})
}

// PKCEVerify validates an S256 code_challenge against a code_verifier.
func PKCEVerify(_ context.Context, challenge, verifier string) bool {
	h := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(h[:])
	return computed == challenge
}
