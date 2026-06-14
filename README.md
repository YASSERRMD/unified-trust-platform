# Unified Trust Platform

A production-grade Enterprise Identity and Access Management (IAM) platform providing SSO, MFA, federation, delegation, just-in-time access, and a multi-model authorization engine (RBAC + ABAC + PBAC).

## Stack

| Layer      | Technology                              |
|------------|-----------------------------------------|
| Backend    | Go 1.22+ (chi router, zap, pgx, redis) |
| Frontend   | Next.js 16 (App Router, TypeScript, Tailwind) |
| Database   | PostgreSQL 16                           |
| Cache      | Redis 7                                 |
| Containers | Docker Compose                          |
| Auth       | OAuth2, OpenID Connect, JWT RS256, PKCE S256 |
| Authz      | RBAC + ABAC + PBAC (deny-override)      |

## Features

| Domain | Capabilities |
|--------|-------------|
| **Tenant Management** | Multi-tenant isolation enforced at every DB query |
| **Users** | bcrypt-12 passwords, policy enforcement, account lockout |
| **OAuth2 / OIDC** | Authorization Code + PKCE S256, refresh, revoke, userinfo, JWKS |
| **MFA** | RFC 6238 TOTP (HMAC-SHA1, 6-digit, 30s), bcrypt-stored secrets |
| **RBAC** | Roles → permissions (resource:action), user-role with expiry |
| **ABAC + PBAC** | Policy documents with conditions, deny-override combining algorithm |
| **Delegation** | Time-bound, scope-bound role delegation between users |
| **JIT Access** | Request → Approve/Deny → time-limited grant workflow |
| **Federation** | OIDC provider config, auto-discovery, callback, identity linking |
| **Audit** | Immutable event log (DB-level rules), cursor pagination, CSV export |
| **Security** | HSTS, CSP, CORS allowlist, per-IP rate limiting, brute force protection |
| **Admin Portal** | Next.js dashboard: tenants, users, roles, policies, audit screens |

## Quick Start

### Prerequisites

- Go 1.22+
- Node.js 20+
- Docker and Docker Compose
- `make`

### 1. Configure environment

```bash
cp .env.example .env
# Required: set DB_PASSWORD
# Optional: set CORS_ALLOWED_ORIGINS (default: http://localhost:3000)
```

### 2. Start infrastructure

```bash
make docker-up
```

This starts PostgreSQL 16 and Redis 7 with health checks.

### 3. Run migrations

```bash
make migrate-up
```

Applies all 11 schema migrations including tenant isolation, RBAC, policies, sessions, MFA, federation, delegation, JIT access, and the immutable audit log.

### 4. Start backend

```bash
make backend
# API available at http://localhost:8080
```

### 5. Start frontend

```bash
make frontend
# Admin portal at http://localhost:3000/admin
```

### Health checks

```
GET http://localhost:8080/health   → { "data": { "status": "ok" } }
GET http://localhost:8080/ready    → { "data": { "db": "ok", "redis": "ok" } }
```

## API Overview

All responses use the standard envelope:

```json
{ "data": { ... }, "meta": { "requestId": "...", "timestamp": "..." } }
```

Errors:
```json
{ "error": { "code": "VALIDATION_FAILED", "message": "...", "details": {} }, "meta": { ... } }
```

Key endpoints:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/.well-known/openid-configuration` | OIDC discovery document |
| `GET` | `/.well-known/jwks.json` | JSON Web Key Set |
| `GET` | `/oauth2/authorize` | Authorization endpoint (PKCE S256 required) |
| `POST` | `/oauth2/token` | Token endpoint |
| `POST` | `/api/v1/tenants` | Create tenant |
| `POST` | `/api/v1/users` | Create user |
| `POST` | `/api/v1/roles` | Create role |
| `POST` | `/api/v1/policies` | Create PBAC policy |
| `POST` | `/api/v1/authz/evaluate` | Evaluate RBAC+ABAC+PBAC decision |
| `POST` | `/api/v1/jit/requests` | Request JIT access |
| `POST` | `/api/v1/delegations` | Create time-bound delegation |
| `GET` | `/api/v1/audit/events` | Query audit log (paginated) |
| `GET` | `/api/v1/audit/events/export` | Export audit log as CSV |

Full reference: [docs/api-design.md](docs/api-design.md) · [docs/openapi.yaml](docs/openapi.yaml)

## Authorization Model

The platform evaluates every access decision in three layers:

1. **RBAC** — User → Role → Permission (`resource:action`)
2. **ABAC** — Policy conditions on subject/resource/environment attributes
3. **PBAC** — Declarative policy documents with priority and effect (`permit|deny`)

**Deny-override**: any explicit `deny` policy beats any `permit`. Implicit deny applies when no policy matches.

```json
POST /api/v1/authz/evaluate
{
  "subject": { "userId": "uuid", "roles": ["admin"] },
  "action": "iam:user:delete",
  "resource": { "type": "user", "id": "uuid" }
}
→ { "allowed": true, "decision": "permit", "reason": "...", "matchedPolicies": [...], "evaluationMs": 2 }
```

## Development Commands

```bash
make help             # list all commands
make test             # run Go tests
make lint             # lint frontend TypeScript
make fmt              # format Go code
make vet              # run go vet
make ci               # fmt + vet + test + build-backend + build-frontend + docker-validate
make migrate-up       # apply all DB migrations
make migrate-down     # roll back last migration
make docker-logs      # follow container logs
```

## Project Structure

```
unified-trust-platform/
├── backend/
│   ├── cmd/api/             # Entrypoint
│   ├── internal/
│   │   ├── audit/           # Immutable audit log
│   │   ├── auth/            # Password hashing and policy
│   │   ├── client/          # OAuth2 client applications
│   │   ├── config/          # Environment-based configuration
│   │   ├── database/        # pgx pool + repository interfaces
│   │   ├── delegation/      # Time-bound role delegation
│   │   ├── federation/      # External OIDC providers
│   │   ├── http/
│   │   │   ├── handler/     # Health handlers
│   │   │   ├── middleware/  # Security, CORS, rate limit, brute force, logging
│   │   │   ├── response/    # Standard JSON envelopes
│   │   │   ├── router/      # chi router wiring
│   │   │   └── validate/    # Request decoding and validation
│   │   ├── jit/             # Just-in-time access requests and grants
│   │   ├── mfa/             # TOTP MFA
│   │   ├── oauth/           # OAuth2/OIDC server
│   │   ├── policy/          # RBAC + ABAC + PBAC engine
│   │   ├── tenant/          # Tenant service
│   │   └── user/            # User service
│   ├── migrations/          # 11 SQL migrations + seed data
│   └── tests/               # Integration and unit tests
├── frontend/
│   ├── app/(admin)/         # Next.js App Router admin pages
│   ├── components/          # Shared UI components
│   └── lib/api.ts           # Typed API client
├── deploy/
│   └── docker-compose.yml   # PostgreSQL + Redis + backend + frontend
├── docs/                    # Architecture, API design, security model
├── Makefile
└── .env.example
```

## Security Notes

- Passwords hashed with bcrypt cost=12; never returned in API responses
- Client secrets hashed with SHA-256 before storage; shown once on creation
- PKCE S256 enforced on all authorization code flows
- Authorization codes: SHA-256 hashed, single-use, 5-minute TTL
- Audit log immutable via PostgreSQL `CREATE RULE no_update_audit` / `no_delete_audit`
- Tenant isolation enforced on every DB query via `tenant_id` column
- Rate limit: 300 req/min globally, 20 req/min on `/oauth2/token`
- Brute force: IP blocked after 10 consecutive 401s in 15 minutes

## Documentation

- [Architecture](docs/architecture.md)
- [Implementation Plan](docs/implementation-plan.md)
- [Security Model](docs/security-model.md)
- [API Design](docs/api-design.md)
- [Authorization Model](docs/authorization-model.md)
- [Git Workflow](docs/git-workflow.md)
- [OpenAPI Spec](docs/openapi.yaml)

## License

MIT — Copyright (c) 2026 YASSERRMD
