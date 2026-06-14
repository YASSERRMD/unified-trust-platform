package user

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YASSERRMD/unified-trust-platform/backend/internal/auth"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/database"
)

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

type CreateInput struct {
	TenantID  uuid.UUID
	Email     string
	Password  string
	FirstName string
	LastName  string
}

type UpdateInput struct {
	FirstName  *string
	LastName   *string
	Department *string
	JobTitle   *string
	Status     *string
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*database.User, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))

	if err := auth.ValidatePasswordPolicy(in.Password, auth.DefaultPolicy); err != nil {
		return nil, err
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}

	u := &database.User{
		ID:           uuid.New(),
		TenantID:     in.TenantID,
		Email:        in.Email,
		PasswordHash: hash,
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO users (id, tenant_id, email, password_hash, status)
		 VALUES ($1, $2, $3, $4, $5)`,
		u.ID, u.TenantID, u.Email, u.PasswordHash, u.Status,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return nil, fmt.Errorf("email already registered in this tenant")
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO user_profiles (user_id, first_name, last_name)
		 VALUES ($1, $2, $3)`,
		u.ID, in.FirstName, in.LastName,
	)
	if err != nil {
		return nil, fmt.Errorf("create user profile: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return u, nil
}

func (s *Service) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*database.User, error) {
	u := &database.User{}
	err := s.db.QueryRow(ctx,
		`SELECT id, tenant_id, email, password_hash, status, email_verified,
		        failed_attempts, locked_until, last_login_at, created_at, updated_at
		 FROM users WHERE id = $1 AND tenant_id = $2 AND status != 'deleted'`,
		id, tenantID,
	).Scan(
		&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Status,
		&u.EmailVerified, &u.FailedAttempts, &u.LockedUntil,
		&u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

func (s *Service) GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*database.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	u := &database.User{}
	err := s.db.QueryRow(ctx,
		`SELECT id, tenant_id, email, password_hash, status, email_verified,
		        failed_attempts, locked_until, last_login_at, created_at, updated_at
		 FROM users WHERE email = $1 AND tenant_id = $2 AND status != 'deleted'`,
		email, tenantID,
	).Scan(
		&u.ID, &u.TenantID, &u.Email, &u.PasswordHash, &u.Status,
		&u.EmailVerified, &u.FailedAttempts, &u.LockedUntil,
		&u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

func (s *Service) List(ctx context.Context, tenantID uuid.UUID, query, status string, limit int, cursor string) ([]*database.User, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	args := []any{tenantID}
	where := "tenant_id = $1 AND status != 'deleted'"
	argN := 2

	if query != "" {
		where += fmt.Sprintf(" AND email ILIKE $%d", argN)
		args = append(args, "%"+query+"%")
		argN++
	}
	if status != "" {
		where += fmt.Sprintf(" AND status = $%d", argN)
		args = append(args, status)
		argN++
	}
	if cursor != "" {
		cid, err := uuid.Parse(cursor)
		if err == nil {
			where += fmt.Sprintf(" AND id > $%d", argN)
			args = append(args, cid)
			argN++
		}
	}

	args = append(args, limit+1)
	q := fmt.Sprintf(`SELECT id, tenant_id, email, status, email_verified, last_login_at, created_at, updated_at
	                  FROM users WHERE %s ORDER BY id LIMIT $%d`, where, argN)

	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, "", fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*database.User
	for rows.Next() {
		u := &database.User{}
		if err := rows.Scan(&u.ID, &u.TenantID, &u.Email, &u.Status, &u.EmailVerified, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, "", err
		}
		users = append(users, u)
	}

	nextCursor := ""
	if len(users) > limit {
		nextCursor = users[limit].ID.String()
		users = users[:limit]
	}

	return users, nextCursor, nil
}

func (s *Service) Update(ctx context.Context, tenantID, id uuid.UUID, in UpdateInput) (*database.User, error) {
	u, err := s.GetByID(ctx, tenantID, id)
	if err != nil || u == nil {
		return nil, err
	}

	if in.Status != nil {
		switch *in.Status {
		case "active", "suspended":
			u.Status = *in.Status
		default:
			return nil, fmt.Errorf("invalid status: must be active or suspended")
		}
		u.UpdatedAt = time.Now().UTC()
		_, err = s.db.Exec(ctx,
			`UPDATE users SET status=$1, updated_at=$2 WHERE id=$3`,
			u.Status, u.UpdatedAt, u.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("update user status: %w", err)
		}
	}

	if in.FirstName != nil || in.LastName != nil || in.Department != nil || in.JobTitle != nil {
		_, err = s.db.Exec(ctx,
			`UPDATE user_profiles
			 SET first_name = COALESCE($1, first_name),
			     last_name  = COALESCE($2, last_name),
			     department = COALESCE($3, department),
			     job_title  = COALESCE($4, job_title),
			     updated_at = NOW()
			 WHERE user_id = $5`,
			in.FirstName, in.LastName, in.Department, in.JobTitle, id,
		)
		if err != nil {
			return nil, fmt.Errorf("update user profile: %w", err)
		}
	}

	return u, nil
}

func (s *Service) SetStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error {
	result, err := s.db.Exec(ctx,
		`UPDATE users SET status=$1, updated_at=NOW() WHERE id=$2 AND tenant_id=$3 AND status != 'deleted'`,
		status, id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("set user status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (s *Service) RecordLogin(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := s.db.Exec(ctx,
		`UPDATE users SET last_login_at=$1, failed_attempts=0, locked_until=NULL WHERE id=$2`,
		now, id,
	)
	return err
}
