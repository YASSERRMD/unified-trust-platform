package policy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Role struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenantId"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ParentID    *uuid.UUID `json:"parentId,omitempty"`
	IsSystem    bool       `json:"isSystem"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type Permission struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenantId"`
	Name        string    `json:"name"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
	IsSystem    bool      `json:"isSystem"`
	CreatedAt   time.Time `json:"createdAt"`
}

type RBACService struct {
	db *pgxpool.Pool
}

func NewRBACService(db *pgxpool.Pool) *RBACService {
	return &RBACService{db: db}
}

func (s *RBACService) CreateRole(ctx context.Context, tenantID uuid.UUID, name, description string) (*Role, error) {
	r := &Role{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        strings.TrimSpace(name),
		Description: description,
		CreatedAt:   time.Now().UTC(),
	}
	_, err := s.db.Exec(ctx,
		`INSERT INTO roles (id, tenant_id, name, description) VALUES ($1, $2, $3, $4)`,
		r.ID, r.TenantID, r.Name, r.Description,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return nil, fmt.Errorf("role %q already exists", name)
		}
		return nil, err
	}
	return r, nil
}

func (s *RBACService) ListRoles(ctx context.Context, tenantID uuid.UUID) ([]*Role, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, tenant_id, name, description, parent_id, is_system, created_at
		 FROM roles WHERE tenant_id = $1 ORDER BY name`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*Role
	for rows.Next() {
		r := &Role{}
		if err := rows.Scan(&r.ID, &r.TenantID, &r.Name, &r.Description, &r.ParentID, &r.IsSystem, &r.CreatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	return roles, nil
}

func (s *RBACService) GetRole(ctx context.Context, id uuid.UUID) (*Role, error) {
	r := &Role{}
	err := s.db.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, parent_id, is_system, created_at FROM roles WHERE id = $1`, id,
	).Scan(&r.ID, &r.TenantID, &r.Name, &r.Description, &r.ParentID, &r.IsSystem, &r.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return r, err
}

func (s *RBACService) CreatePermission(ctx context.Context, tenantID uuid.UUID, name, resource, action, description string) (*Permission, error) {
	p := &Permission{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        name,
		Resource:    resource,
		Action:      action,
		Description: description,
		CreatedAt:   time.Now().UTC(),
	}
	_, err := s.db.Exec(ctx,
		`INSERT INTO permissions (id, tenant_id, name, resource, action, description) VALUES ($1, $2, $3, $4, $5, $6)`,
		p.ID, p.TenantID, p.Name, p.Resource, p.Action, p.Description,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *RBACService) ListPermissions(ctx context.Context, tenantID uuid.UUID) ([]*Permission, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, tenant_id, name, resource, action, description, is_system, created_at
		 FROM permissions WHERE tenant_id = $1 ORDER BY resource, action`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []*Permission
	for rows.Next() {
		p := &Permission{}
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Resource, &p.Action, &p.Description, &p.IsSystem, &p.CreatedAt); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, nil
}

func (s *RBACService) AssignRolePermission(ctx context.Context, roleID, permID uuid.UUID) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		roleID, permID,
	)
	return err
}

func (s *RBACService) AssignUserRole(ctx context.Context, userID, roleID, tenantID uuid.UUID, grantedBy *uuid.UUID, expiresAt *time.Time) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id, tenant_id, granted_by, expires_at)
		 VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`,
		userID, roleID, tenantID, grantedBy, expiresAt,
	)
	return err
}

func (s *RBACService) RemoveUserRole(ctx context.Context, userID, roleID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2`, userID, roleID)
	return err
}

// GetEffectivePermissions returns the flat set of permission strings for a user in a tenant.
func (s *RBACService) GetEffectivePermissions(ctx context.Context, userID, tenantID uuid.UUID) (map[string]bool, error) {
	rows, err := s.db.Query(ctx,
		`SELECT DISTINCT p.resource || ':' || p.action
		 FROM user_roles ur
		 JOIN roles r ON r.id = ur.role_id
		 JOIN role_permissions rp ON rp.role_id = r.id
		 JOIN permissions p ON p.id = rp.permission_id
		 WHERE ur.user_id = $1 AND ur.tenant_id = $2
		   AND (ur.expires_at IS NULL OR ur.expires_at > NOW())`,
		userID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	perms := map[string]bool{}
	for rows.Next() {
		var perm string
		if err := rows.Scan(&perm); err != nil {
			return nil, err
		}
		perms[perm] = true
	}
	return perms, nil
}

// HasPermission checks if userID has resource:action in tenantID.
func (s *RBACService) HasPermission(ctx context.Context, userID, tenantID uuid.UUID, resource, action string) (bool, error) {
	perms, err := s.GetEffectivePermissions(ctx, userID, tenantID)
	if err != nil {
		return false, err
	}
	return perms[resource+":"+action], nil
}
