package jit

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusDenied   = "denied"
	StatusExpired  = "expired"
	StatusRevoked  = "revoked"
)

type Request struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenantId"`
	RequesterID   uuid.UUID  `json:"requesterId"`
	ResourceType  string     `json:"resourceType"`
	ResourceID    string     `json:"resourceId"`
	RequestedRole string     `json:"requestedRole"`
	Justification string     `json:"justification"`
	Status        string     `json:"status"`
	RequestedDur  int        `json:"requestedDurationMinutes"`
	CreatedAt     time.Time  `json:"createdAt"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
}

type Grant struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenantId"`
	RequestID   uuid.UUID  `json:"requestId"`
	RequesterID uuid.UUID  `json:"requesterId"`
	ApproverID  uuid.UUID  `json:"approverId"`
	GrantedRole string     `json:"grantedRole"`
	StartsAt    time.Time  `json:"startsAt"`
	ExpiresAt   time.Time  `json:"expiresAt"`
	RevokedAt   *time.Time `json:"revokedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) CreateRequest(ctx context.Context, tenantID, requesterID uuid.UUID, resourceType, resourceID, requestedRole, justification string, durationMinutes int) (*Request, error) {
	if durationMinutes <= 0 || durationMinutes > 480 {
		return nil, fmt.Errorf("requestedDurationMinutes must be between 1 and 480")
	}
	if justification == "" {
		return nil, fmt.Errorf("justification is required")
	}

	req := &Request{
		ID:            uuid.New(),
		TenantID:      tenantID,
		RequesterID:   requesterID,
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		RequestedRole: requestedRole,
		Justification: justification,
		Status:        StatusPending,
		RequestedDur:  durationMinutes,
		CreatedAt:     time.Now().UTC(),
	}

	_, err := s.db.Exec(ctx,
		`INSERT INTO jit_requests (id, tenant_id, requester_id, resource_type, resource_id, requested_role, justification, status, requested_duration_minutes)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		req.ID, req.TenantID, req.RequesterID, req.ResourceType, req.ResourceID,
		req.RequestedRole, req.Justification, req.Status, req.RequestedDur,
	)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (s *Service) GetRequest(ctx context.Context, tenantID, id uuid.UUID) (*Request, error) {
	req := &Request{}
	err := s.db.QueryRow(ctx,
		`SELECT id, tenant_id, requester_id, resource_type, resource_id, requested_role, justification, status, requested_duration_minutes, created_at, expires_at
		 FROM jit_requests WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	).Scan(&req.ID, &req.TenantID, &req.RequesterID, &req.ResourceType, &req.ResourceID,
		&req.RequestedRole, &req.Justification, &req.Status, &req.RequestedDur, &req.CreatedAt, &req.ExpiresAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return req, err
}

func (s *Service) ListRequests(ctx context.Context, tenantID uuid.UUID, status string) ([]*Request, error) {
	q := `SELECT id, tenant_id, requester_id, resource_type, resource_id, requested_role, justification, status, requested_duration_minutes, created_at, expires_at
	      FROM jit_requests WHERE tenant_id = $1`
	args := []any{tenantID}
	if status != "" {
		q += " AND status = $2"
		args = append(args, status)
	}
	q += " ORDER BY created_at DESC"

	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*Request
	for rows.Next() {
		req := &Request{}
		if err := rows.Scan(&req.ID, &req.TenantID, &req.RequesterID, &req.ResourceType, &req.ResourceID,
			&req.RequestedRole, &req.Justification, &req.Status, &req.RequestedDur, &req.CreatedAt, &req.ExpiresAt); err != nil {
			return nil, err
		}
		results = append(results, req)
	}
	return results, nil
}

// Approve approves a JIT request and creates a time-limited grant.
func (s *Service) Approve(ctx context.Context, tenantID, requestID, approverID uuid.UUID) (*Grant, error) {
	req, err := s.GetRequest(ctx, tenantID, requestID)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("request not found")
	}
	if req.Status != StatusPending {
		return nil, fmt.Errorf("request is not pending (status: %s)", req.Status)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(req.RequestedDur) * time.Minute)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`UPDATE jit_requests SET status = $1, expires_at = $2 WHERE id = $3`,
		StatusApproved, expiresAt, requestID,
	)
	if err != nil {
		return nil, err
	}

	grant := &Grant{
		ID:          uuid.New(),
		TenantID:    tenantID,
		RequestID:   requestID,
		RequesterID: req.RequesterID,
		ApproverID:  approverID,
		GrantedRole: req.RequestedRole,
		StartsAt:    now,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO jit_grants (id, tenant_id, request_id, requester_id, approver_id, granted_role, starts_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		grant.ID, grant.TenantID, grant.RequestID, grant.RequesterID, grant.ApproverID,
		grant.GrantedRole, grant.StartsAt, grant.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return grant, nil
}

// Deny rejects a JIT request.
func (s *Service) Deny(ctx context.Context, tenantID, requestID uuid.UUID) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE jit_requests SET status = $1 WHERE id = $2 AND tenant_id = $3 AND status = $4`,
		StatusDenied, requestID, tenantID, StatusPending,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("request not found or not pending")
	}
	return nil
}

// RevokeGrant revokes an active JIT grant before it expires.
func (s *Service) RevokeGrant(ctx context.Context, tenantID, grantID uuid.UUID) error {
	now := time.Now().UTC()
	tag, err := s.db.Exec(ctx,
		`UPDATE jit_grants SET revoked_at = $1 WHERE id = $2 AND tenant_id = $3 AND revoked_at IS NULL`,
		now, grantID, tenantID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("grant not found or already revoked")
	}
	return nil
}

// GetActiveGrants returns grants that are currently valid for a user.
func (s *Service) GetActiveGrants(ctx context.Context, tenantID, requesterID uuid.UUID) ([]*Grant, error) {
	now := time.Now().UTC()
	rows, err := s.db.Query(ctx,
		`SELECT id, tenant_id, request_id, requester_id, approver_id, granted_role, starts_at, expires_at, revoked_at, created_at
		 FROM jit_grants
		 WHERE tenant_id = $1 AND requester_id = $2
		   AND starts_at <= $3 AND expires_at > $3 AND revoked_at IS NULL
		 ORDER BY expires_at`,
		tenantID, requesterID, now,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var grants []*Grant
	for rows.Next() {
		g := &Grant{}
		if err := rows.Scan(&g.ID, &g.TenantID, &g.RequestID, &g.RequesterID, &g.ApproverID,
			&g.GrantedRole, &g.StartsAt, &g.ExpiresAt, &g.RevokedAt, &g.CreatedAt); err != nil {
			return nil, err
		}
		grants = append(grants, g)
	}
	return grants, nil
}
