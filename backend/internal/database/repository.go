package database

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Tenant repository

type Tenant struct {
	ID        uuid.UUID      `json:"id"`
	Name      string         `json:"name"`
	Slug      string         `json:"slug"`
	Status    string         `json:"status"`
	Settings  map[string]any `json:"settings"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

type TenantRepository interface {
	Create(ctx context.Context, t *Tenant) error
	GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*Tenant, error)
	List(ctx context.Context, limit int, cursor string) ([]*Tenant, string, error)
	Update(ctx context.Context, t *Tenant) error
}

// User repository

type User struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenantId"`
	Email          string     `json:"email"`
	PasswordHash   string     `json:"-"`
	Status         string     `json:"status"`
	EmailVerified  bool       `json:"emailVerified"`
	FailedAttempts int        `json:"-"`
	LockedUntil    *time.Time `json:"-"`
	LastLoginAt    *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type UserRepository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*User, error)
	List(ctx context.Context, tenantID uuid.UUID, query string, status string, limit int, cursor string) ([]*User, string, error)
	Update(ctx context.Context, u *User) error
	IncrementFailedAttempts(ctx context.Context, id uuid.UUID) error
	ResetFailedAttempts(ctx context.Context, id uuid.UUID) error
}

// Role repository

type Role struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenantId"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ParentID    *uuid.UUID `json:"parentId,omitempty"`
	IsSystem    bool       `json:"isSystem"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
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

type RoleRepository interface {
	Create(ctx context.Context, r *Role) error
	GetByID(ctx context.Context, id uuid.UUID) (*Role, error)
	List(ctx context.Context, tenantID uuid.UUID, limit int, cursor string) ([]*Role, string, error)
	AssignPermission(ctx context.Context, roleID, permID uuid.UUID) error
	RemovePermission(ctx context.Context, roleID, permID uuid.UUID) error
	GetPermissions(ctx context.Context, roleID uuid.UUID) ([]*Permission, error)
}

type PermissionRepository interface {
	Create(ctx context.Context, p *Permission) error
	GetByID(ctx context.Context, id uuid.UUID) (*Permission, error)
	List(ctx context.Context, tenantID uuid.UUID, limit int, cursor string) ([]*Permission, string, error)
}

type UserRoleRepository interface {
	Assign(ctx context.Context, userID, roleID, tenantID uuid.UUID, grantedBy *uuid.UUID, expiresAt *time.Time) error
	Remove(ctx context.Context, userID, roleID uuid.UUID) error
	GetRolesForUser(ctx context.Context, userID, tenantID uuid.UUID) ([]*Role, error)
}

// AuditEvent repository

type AuditEvent struct {
	ID           uuid.UUID      `json:"id"`
	TenantID     *uuid.UUID     `json:"tenantId,omitempty"`
	ActorID      *uuid.UUID     `json:"actorId,omitempty"`
	ActorEmail   string         `json:"actorEmail,omitempty"`
	Category     string         `json:"category"`
	EventType    string         `json:"eventType"`
	ResourceType string         `json:"resourceType,omitempty"`
	ResourceID   string         `json:"resourceId,omitempty"`
	Outcome      string         `json:"outcome"`
	IPAddress    string         `json:"ipAddress,omitempty"`
	UserAgent    string         `json:"userAgent,omitempty"`
	Metadata     map[string]any `json:"metadata"`
	OccurredAt   time.Time      `json:"occurredAt"`
}

type AuditRepository interface {
	Write(ctx context.Context, e *AuditEvent) error
	GetByID(ctx context.Context, id uuid.UUID) (*AuditEvent, error)
	List(ctx context.Context, tenantID *uuid.UUID, category, eventType string, limit int, cursor string) ([]*AuditEvent, string, error)
}
