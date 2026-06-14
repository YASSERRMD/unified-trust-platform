package delegation

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Delegation struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenantId"`
	DelegatorID  uuid.UUID  `json:"delegatorId"`
	DelegateID   uuid.UUID  `json:"delegateId"`
	ScopeType    string     `json:"scopeType"`
	GrantedRoles []string   `json:"grantedRoles"`
	Reason       string     `json:"reason"`
	Status       string     `json:"status"`
	StartsAt     time.Time  `json:"startsAt"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	RevokedAt    *time.Time `json:"revokedAt,omitempty"`
	RevokedBy    *uuid.UUID `json:"revokedBy,omitempty"`
}

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) Create(ctx context.Context, tenantID, delegatorID, delegateID uuid.UUID, scopeType string, grantedRoles []string, reason string, startsAt time.Time, expiresAt *time.Time) (*Delegation, error) {
	if len(grantedRoles) == 0 {
		return nil, fmt.Errorf("at least one role must be granted")
	}
	if !startsAt.IsZero() && expiresAt != nil && expiresAt.Before(startsAt) {
		return nil, fmt.Errorf("expiresAt must be after startsAt")
	}
	if startsAt.IsZero() {
		startsAt = time.Now().UTC()
	}

	d := &Delegation{
		ID:           uuid.New(),
		TenantID:     tenantID,
		DelegatorID:  delegatorID,
		DelegateID:   delegateID,
		ScopeType:    scopeType,
		GrantedRoles: grantedRoles,
		Reason:       reason,
		Status:       "active",
		StartsAt:     startsAt,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now().UTC(),
	}

	_, err := s.db.Exec(ctx,
		`INSERT INTO delegations (id, tenant_id, delegator_id, delegate_id, scope_type, granted_roles, reason, status, starts_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		d.ID, d.TenantID, d.DelegatorID, d.DelegateID, d.ScopeType, d.GrantedRoles, d.Reason, d.Status, d.StartsAt, d.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*Delegation, error) {
	d := &Delegation{}
	err := s.db.QueryRow(ctx,
		`SELECT id, tenant_id, delegator_id, delegate_id, scope_type, granted_roles, reason, status, starts_at, expires_at, created_at, revoked_at, revoked_by
		 FROM delegations WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	).Scan(&d.ID, &d.TenantID, &d.DelegatorID, &d.DelegateID, &d.ScopeType, &d.GrantedRoles,
		&d.Reason, &d.Status, &d.StartsAt, &d.ExpiresAt, &d.CreatedAt, &d.RevokedAt, &d.RevokedBy)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return d, err
}

func (s *Service) ListByDelegator(ctx context.Context, tenantID, delegatorID uuid.UUID) ([]*Delegation, error) {
	return s.list(ctx, tenantID, "delegator_id", delegatorID)
}

func (s *Service) ListByDelegate(ctx context.Context, tenantID, delegateID uuid.UUID) ([]*Delegation, error) {
	return s.list(ctx, tenantID, "delegate_id", delegateID)
}

func (s *Service) list(ctx context.Context, tenantID uuid.UUID, col string, userID uuid.UUID) ([]*Delegation, error) {
	rows, err := s.db.Query(ctx,
		fmt.Sprintf(`SELECT id, tenant_id, delegator_id, delegate_id, scope_type, granted_roles, reason, status, starts_at, expires_at, created_at, revoked_at, revoked_by
		 FROM delegations WHERE tenant_id = $1 AND %s = $2 ORDER BY created_at DESC`, col),
		tenantID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*Delegation
	for rows.Next() {
		d := &Delegation{}
		if err := rows.Scan(&d.ID, &d.TenantID, &d.DelegatorID, &d.DelegateID, &d.ScopeType, &d.GrantedRoles,
			&d.Reason, &d.Status, &d.StartsAt, &d.ExpiresAt, &d.CreatedAt, &d.RevokedAt, &d.RevokedBy); err != nil {
			return nil, err
		}
		results = append(results, d)
	}
	return results, nil
}

// Revoke marks a delegation as revoked. Only the delegator or a tenant admin may revoke.
func (s *Service) Revoke(ctx context.Context, tenantID, id, revokedBy uuid.UUID) error {
	now := time.Now().UTC()
	tag, err := s.db.Exec(ctx,
		`UPDATE delegations SET status='revoked', revoked_at=$1, revoked_by=$2
		 WHERE id=$3 AND tenant_id=$4 AND status='active'`,
		now, revokedBy, id, tenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delegation not found or already revoked")
	}
	return nil
}

// GetEffectiveDelegations returns active, currently-valid delegations for a delegate user.
func (s *Service) GetEffectiveDelegations(ctx context.Context, tenantID, delegateID uuid.UUID) ([]*Delegation, error) {
	now := time.Now().UTC()
	rows, err := s.db.Query(ctx,
		`SELECT id, tenant_id, delegator_id, delegate_id, scope_type, granted_roles, reason, status, starts_at, expires_at, created_at, revoked_at, revoked_by
		 FROM delegations
		 WHERE tenant_id = $1 AND delegate_id = $2 AND status = 'active'
		   AND starts_at <= $3
		   AND (expires_at IS NULL OR expires_at > $3)
		 ORDER BY starts_at`,
		tenantID, delegateID, now,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*Delegation
	for rows.Next() {
		d := &Delegation{}
		if err := rows.Scan(&d.ID, &d.TenantID, &d.DelegatorID, &d.DelegateID, &d.ScopeType, &d.GrantedRoles,
			&d.Reason, &d.Status, &d.StartsAt, &d.ExpiresAt, &d.CreatedAt, &d.RevokedAt, &d.RevokedBy); err != nil {
			return nil, err
		}
		results = append(results, d)
	}
	return results, nil
}
