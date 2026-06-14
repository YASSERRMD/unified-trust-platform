# Unified Trust Platform — Implementation Plan

## Repository State at Phase 0

- Single file: `LICENSE` (MIT)
- No existing backend, frontend, or infrastructure code
- Clean slate — full build from scratch

---

## Phase Summary

| Phase | Name                          | Status  | Branch              |
|-------|-------------------------------|---------|---------------------|
| 0     | Repository Inspection & Plan  | Current | phase-00-planning   |
| 1     | Project Foundation            | Pending | phase-01-foundation |
| 2     | Core Data Model               | Pending | phase-02-data-model |
| 3     | Backend API Foundation        | Pending | phase-03-api-foundation |
| 4     | Tenant & User Management      | Pending | phase-04-tenant-user |
| 5     | OAuth2 & OIDC Foundation      | Pending | phase-05-oauth-oidc |
| 6     | MFA Foundation                | Pending | phase-06-mfa        |
| 7     | RBAC Authorization            | Pending | phase-07-rbac       |
| 8     | ABAC & PBAC Policy Engine     | Pending | phase-08-policy     |
| 9     | Delegation                    | Pending | phase-09-delegation |
| 10    | Just-in-Time Access           | Pending | phase-10-jit        |
| 11    | Federation Foundation         | Pending | phase-11-federation |
| 12    | Audit & Compliance Center     | Pending | phase-12-audit      |
| 13    | Frontend Foundation           | Pending | phase-13-frontend   |
| 14    | Frontend IAM Screens          | Pending | phase-14-iam-screens |
| 15    | Security Hardening            | Pending | phase-15-security   |
| 16    | Developer Experience & Docs   | Pending | phase-16-dx-docs    |

---

## Phase 0: Repository Inspection and Planning
**Branch:** `phase-00-planning`
**Goal:** Document the current state and establish the implementation plan.

### Atomic Tasks
- [x] Inspect repository structure
- [x] Identify existing files (LICENSE only)
- [x] Identify package managers and frameworks (none yet)
- [x] Create `docs/architecture.md`
- [x] Create `docs/implementation-plan.md`
- [x] Create `docs/security-model.md`
- [x] Create `docs/api-design.md`
- [x] Create `docs/authorization-model.md`
- [x] Create `docs/git-workflow.md`

---

## Phase 1: Project Foundation
**Branch:** `phase-01-foundation`
**Goal:** Bootstrap the Go backend, Next.js frontend, Docker Compose, and make both start successfully.

### Atomic Tasks
1. Initialize Go module (`backend/go.mod`)
2. Add backend directory structure (`cmd/`, `internal/`, `migrations/`, `tests/`)
3. Add `cmd/api/main.go` with health endpoint
4. Initialize Next.js frontend (`frontend/`)
5. Add backend `Dockerfile`
6. Add frontend `Dockerfile`
7. Add `deploy/docker-compose.yml` with PostgreSQL and Redis
8. Add `.env.example` files
9. Add root `Makefile` with dev commands
10. Add root `README.md`
11. Verify `go build ./...` passes
12. Verify `npm run build` passes

### Verification
```bash
go build ./...
go vet ./...
npm run lint
npm run build
docker compose config
```

---

## Phase 2: Core Data Model
**Branch:** `phase-02-data-model`
**Goal:** Define all database schemas via migrations and repository interfaces.

### Entities
- Tenant, User, UserProfile
- ClientApplication, RedirectUri
- Role, Permission, UserRole, RolePermission
- Policy, PolicyRule, PolicyAttribute
- Session, Token
- MfaMethod, MfaChallenge
- FederationProvider, LinkedIdentity
- Delegation
- JitAccessRequest, JitGrant
- AuditEvent

### Atomic Tasks
1. Add `golang-migrate` dependency
2. `001_tenants.up.sql`
3. `002_users.up.sql`
4. `003_user_profiles.up.sql`
5. `004_client_applications.up.sql`
6. `005_redirect_uris.up.sql`
7. `006_roles_permissions.up.sql`
8. `007_user_roles.up.sql`
9. `008_policies.up.sql`
10. `009_sessions_tokens.up.sql`
11. `010_mfa.up.sql`
12. `011_federation.up.sql`
13. `012_delegations.up.sql`
14. `013_jit_access.up.sql`
15. `014_audit_events.up.sql`
16. Add repository interfaces in `internal/database/`
17. Add seed data script

---

## Phase 3: Backend API Foundation
**Branch:** `phase-03-api-foundation`
**Goal:** HTTP router, middleware stack, config, logging, and base endpoints.

### Atomic Tasks
1. Add `chi` router dependency
2. Add config loader using env vars
3. Add `zap` structured logging
4. Add request-ID middleware
5. Add standard error response model
6. Add validation helpers
7. `GET /health` endpoint
8. `GET /ready` endpoint (DB + Redis ping)
9. API version prefix `/api/v1/`
10. OpenAPI skeleton (`docs/openapi.yaml`)

---

## Phase 4: Tenant & User Management
**Branch:** `phase-04-tenant-user`

### Endpoints
```
POST   /api/v1/tenants
GET    /api/v1/tenants
GET    /api/v1/tenants/{id}
POST   /api/v1/users
GET    /api/v1/users
GET    /api/v1/users/{id}
PATCH  /api/v1/users/{id}
```

---

## Phase 5: OAuth2 & OIDC Foundation
**Branch:** `phase-05-oauth-oidc`

### Endpoints
```
GET  /.well-known/openid-configuration
GET  /.well-known/jwks.json
GET  /oauth2/authorize
POST /oauth2/token
POST /oauth2/revoke
GET  /oauth2/userinfo
POST /oauth2/logout
```

### Key Capabilities
- RS256 JWT signing with key rotation
- PKCE (S256 only) enforcement
- Authorization code with 5-minute expiry
- Access token (15 min), Refresh token (7 days)
- ID token with standard OIDC claims

---

## Phase 6: MFA Foundation
**Branch:** `phase-06-mfa`

### Endpoints
```
POST /api/v1/mfa/enroll
POST /api/v1/mfa/challenge
POST /api/v1/mfa/verify
GET  /api/v1/mfa/methods
```

---

## Phase 7: RBAC Authorization
**Branch:** `phase-07-rbac`

### Endpoints
```
POST /api/v1/roles
GET  /api/v1/roles
POST /api/v1/permissions
GET  /api/v1/permissions
POST /api/v1/users/{id}/roles
POST /api/v1/roles/{id}/permissions
POST /api/v1/authz/evaluate
```

---

## Phase 8: ABAC & PBAC Policy Engine
**Branch:** `phase-08-policy`

### Decision Response
```json
{
  "allowed": true,
  "decision": "permit",
  "reason": "Role Admin matches; department attribute matches resource owner",
  "matchedPolicies": ["policy-uuid-1"]
}
```

---

## Phase 9: Delegation
**Branch:** `phase-09-delegation`

### Endpoints
```
POST /api/v1/delegations
GET  /api/v1/delegations
POST /api/v1/delegations/{id}/approve
POST /api/v1/delegations/{id}/revoke
```

---

## Phase 10: Just-in-Time Access
**Branch:** `phase-10-jit`

### Endpoints
```
POST /api/v1/jit/requests
GET  /api/v1/jit/requests
POST /api/v1/jit/requests/{id}/approve
POST /api/v1/jit/requests/{id}/reject
POST /api/v1/jit/requests/{id}/revoke
```

---

## Phase 11: Federation Foundation
**Branch:** `phase-11-federation`

### Endpoints
```
POST /api/v1/federation/providers
GET  /api/v1/federation/providers
POST /api/v1/federation/providers/{id}/test
GET  /federation/{provider}/login
GET  /federation/{provider}/callback
```

---

## Phase 12: Audit & Compliance
**Branch:** `phase-12-audit`

### Endpoints
```
GET /api/v1/audit/events
GET /api/v1/audit/events/{id}
GET /api/v1/audit/export
```

---

## Phase 13: Frontend Foundation
**Branch:** `phase-13-frontend`

### Pages
```
/dashboard
/tenants
/users
/applications
/roles
/policies
/delegations
/jit-access
/federation
/audit
```

---

## Phase 14: Frontend IAM Screens
**Branch:** `phase-14-iam-screens`

---

## Phase 15: Security Hardening
**Branch:** `phase-15-security`

### Controls
- Secure HTTP headers (HSTS, CSP, X-Frame-Options, X-Content-Type)
- CORS allowlist
- Rate limiting (token bucket, per-IP)
- Brute-force lockout (5 failed attempts → 15-minute lockout)
- Password policy (min 12 chars, complexity, breach detection hook)
- Token expiry configuration via env

---

## Phase 16: Developer Experience & Docs
**Branch:** `phase-16-dx-docs`

---

## Development Rules

### Git
- Branch per phase: `phase-XX-short-name`
- Atomic commits: one small completed task per commit
- Commit format: `phase <N>: <action>`
- Author: `YASSERRMD <arafath.yasser@gmail.com>`
- Merge each phase into `main` via PR, delete branch

### Verification (run after every phase)
```bash
# Backend
go fmt ./...
go test ./...
go vet ./...

# Frontend
npm run lint
npm run build

# Infrastructure
docker compose config
```

### Commit Author Configuration
```bash
git config user.name "YASSERRMD"
git config user.email "arafath.yasser@gmail.com"
```
