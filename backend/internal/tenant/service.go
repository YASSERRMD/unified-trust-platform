package tenant

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/YASSERRMD/unified-trust-platform/backend/internal/database"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

type CreateInput struct {
	Name string
	Slug string
}

type UpdateInput struct {
	Name   *string
	Status *string
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*database.Tenant, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Slug = strings.TrimSpace(strings.ToLower(in.Slug))

	if in.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if !slugPattern.MatchString(in.Slug) {
		return nil, fmt.Errorf("slug must contain only lowercase letters, digits, and hyphens")
	}

	t := &database.Tenant{
		ID:        uuid.New(),
		Name:      in.Name,
		Slug:      in.Slug,
		Status:    "active",
		Settings:  map[string]any{},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	_, err := s.db.Exec(ctx,
		`INSERT INTO tenants (id, name, slug, status, settings)
		 VALUES ($1, $2, $3, $4, $5)`,
		t.ID, t.Name, t.Slug, t.Status, t.Settings,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return nil, fmt.Errorf("tenant slug already exists")
		}
		return nil, fmt.Errorf("create tenant: %w", err)
	}

	return t, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*database.Tenant, error) {
	t := &database.Tenant{}
	err := s.db.QueryRow(ctx,
		`SELECT id, name, slug, status, settings, created_at, updated_at
		 FROM tenants WHERE id = $1 AND status != 'deleted'`,
		id,
	).Scan(&t.ID, &t.Name, &t.Slug, &t.Status, &t.Settings, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}
	return t, nil
}

func (s *Service) List(ctx context.Context, limit int, cursor string) ([]*database.Tenant, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var rows pgx.Rows
	var err error

	if cursor != "" {
		cursorID, parseErr := uuid.Parse(cursor)
		if parseErr != nil {
			return nil, "", fmt.Errorf("invalid cursor")
		}
		rows, err = s.db.Query(ctx,
			`SELECT id, name, slug, status, settings, created_at, updated_at
			 FROM tenants WHERE status != 'deleted' AND id > $1
			 ORDER BY id LIMIT $2`,
			cursorID, limit+1,
		)
	} else {
		rows, err = s.db.Query(ctx,
			`SELECT id, name, slug, status, settings, created_at, updated_at
			 FROM tenants WHERE status != 'deleted'
			 ORDER BY id LIMIT $1`,
			limit+1,
		)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*database.Tenant
	for rows.Next() {
		t := &database.Tenant{}
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Status, &t.Settings, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, "", err
		}
		tenants = append(tenants, t)
	}

	nextCursor := ""
	if len(tenants) > limit {
		nextCursor = tenants[limit].ID.String()
		tenants = tenants[:limit]
	}

	return tenants, nextCursor, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (*database.Tenant, error) {
	t, err := s.GetByID(ctx, id)
	if err != nil || t == nil {
		return nil, err
	}

	if in.Name != nil {
		t.Name = strings.TrimSpace(*in.Name)
	}
	if in.Status != nil {
		switch *in.Status {
		case "active", "suspended":
			t.Status = *in.Status
		default:
			return nil, fmt.Errorf("invalid status")
		}
	}

	t.UpdatedAt = time.Now().UTC()
	_, err = s.db.Exec(ctx,
		`UPDATE tenants SET name=$1, status=$2, updated_at=$3 WHERE id=$4`,
		t.Name, t.Status, t.UpdatedAt, t.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update tenant: %w", err)
	}

	return t, nil
}
