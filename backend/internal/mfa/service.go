package mfa

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YASSERRMD/unified-trust-platform/backend/internal/auth"
)

type MFAMethod struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"userId"`
	TenantID   uuid.UUID `json:"tenantId"`
	MethodType string    `json:"methodType"`
	Name       string    `json:"name"`
	IsPrimary  bool      `json:"isPrimary"`
	IsVerified bool      `json:"isVerified"`
	CreatedAt  time.Time `json:"createdAt"`
}

type Service struct {
	db        *pgxpool.Pool
	providers map[string]Provider
}

func NewService(db *pgxpool.Pool, issuer string) *Service {
	return &Service{
		db: db,
		providers: map[string]Provider{
			"totp": NewTOTPProvider(issuer),
		},
	}
}

func (s *Service) Enroll(ctx context.Context, userID, tenantID uuid.UUID, methodType, name string) (*EnrollResult, *MFAMethod, error) {
	p, ok := s.providers[methodType]
	if !ok {
		return nil, nil, fmt.Errorf("unsupported MFA method: %s", methodType)
	}

	result, err := p.Enroll(ctx, userID.String(), tenantID.String())
	if err != nil {
		return nil, nil, err
	}

	secretHash := ""
	if result.Secret != "" {
		h, err := auth.HashPassword(result.Secret)
		if err != nil {
			return nil, nil, err
		}
		secretHash = h
	}

	m := &MFAMethod{
		ID:         uuid.New(),
		UserID:     userID,
		TenantID:   tenantID,
		MethodType: methodType,
		Name:       name,
		IsPrimary:  false,
		IsVerified: false,
		CreatedAt:  time.Now().UTC(),
	}

	_, err = s.db.Exec(ctx,
		`INSERT INTO mfa_methods (id, user_id, tenant_id, method_type, secret_hash, name, is_primary, is_verified)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		m.ID, m.UserID, m.TenantID, m.MethodType, secretHash, m.Name, m.IsPrimary, m.IsVerified,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("enroll mfa: %w", err)
	}

	return result, m, nil
}

func (s *Service) Challenge(ctx context.Context, userID uuid.UUID, methodType string) (string, error) {
	var methodID uuid.UUID
	var secretHash string
	err := s.db.QueryRow(ctx,
		`SELECT id, secret_hash FROM mfa_methods
		 WHERE user_id = $1 AND method_type = $2 AND is_verified = TRUE`,
		userID, methodType,
	).Scan(&methodID, &secretHash)
	if err == pgx.ErrNoRows {
		return "", fmt.Errorf("no verified MFA method found")
	}
	if err != nil {
		return "", err
	}

	challengeID := uuid.New()
	var otpHash string

	if methodType != "totp" {
		otp, err := generateOTP()
		if err != nil {
			return "", err
		}
		h := sha256.Sum256([]byte(otp))
		otpHash = hex.EncodeToString(h[:])
	}

	_, err = s.db.Exec(ctx,
		`INSERT INTO mfa_challenges (id, user_id, method_id, method_type, otp_hash, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		challengeID, userID, methodID, methodType, otpHash, time.Now().UTC().Add(5*time.Minute),
	)
	if err != nil {
		return "", fmt.Errorf("create challenge: %w", err)
	}

	return challengeID.String(), nil
}

func (s *Service) Verify(ctx context.Context, userID uuid.UUID, challengeID, code string) (bool, error) {
	cid, err := uuid.Parse(challengeID)
	if err != nil {
		return false, fmt.Errorf("invalid challenge id")
	}

	var methodType, secretHash, otpHash string
	var attempts int
	var methodID uuid.UUID
	err = s.db.QueryRow(ctx,
		`SELECT c.method_type, c.otp_hash, c.attempts, c.method_id, m.secret_hash
		 FROM mfa_challenges c
		 JOIN mfa_methods m ON m.id = c.method_id
		 WHERE c.id = $1 AND c.user_id = $2 AND c.is_used = FALSE AND c.expires_at > NOW()`,
		cid, userID,
	).Scan(&methodType, &otpHash, &attempts, &methodID, &secretHash)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if attempts >= 5 {
		return false, fmt.Errorf("too many attempts")
	}

	s.db.Exec(ctx, `UPDATE mfa_challenges SET attempts = attempts + 1 WHERE id = $1`, cid)

	p, ok := s.providers[methodType]
	if !ok {
		return false, fmt.Errorf("unknown method type")
	}

	ok2, err := p.Verify(ctx, cid.String(), code, secretHash)
	if err != nil {
		return false, err
	}

	if ok2 {
		s.db.Exec(ctx, `UPDATE mfa_challenges SET is_used = TRUE WHERE id = $1`, cid)
	}

	return ok2, nil
}

func (s *Service) ListMethods(ctx context.Context, userID uuid.UUID) ([]*MFAMethod, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, user_id, tenant_id, method_type, name, is_primary, is_verified, created_at
		 FROM mfa_methods WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var methods []*MFAMethod
	for rows.Next() {
		m := &MFAMethod{}
		if err := rows.Scan(&m.ID, &m.UserID, &m.TenantID, &m.MethodType, &m.Name, &m.IsPrimary, &m.IsVerified, &m.CreatedAt); err != nil {
			return nil, err
		}
		methods = append(methods, m)
	}
	return methods, nil
}

func generateOTP() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	n := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	return fmt.Sprintf("%06d", n%1000000), nil
}
