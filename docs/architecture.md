# Unified Trust Platform — Architecture

## Overview

The Unified Trust Platform (UTP) is a production-grade Enterprise Identity and Access Management (IAM) ecosystem. It provides SSO, MFA, federation, delegation, just-in-time access, and a multi-model authorization engine.

---

## System Components

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Unified Trust Platform                        │
│                                                                      │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────────┐  │
│  │  Next.js     │    │  Go Backend  │    │  Policy Engine       │  │
│  │  Admin UI    │───▶│  REST API    │───▶│  RBAC + ABAC + PBAC  │  │
│  └──────────────┘    └──────┬───────┘    └──────────────────────┘  │
│                             │                                        │
│              ┌──────────────┼──────────────┐                        │
│              ▼              ▼              ▼                        │
│       ┌──────────┐  ┌──────────┐  ┌──────────────┐                │
│       │PostgreSQL│  │  Redis   │  │ NATS/Kafka   │                │
│       │ Database │  │  Cache   │  │  Events      │                │
│       └──────────┘  └──────────┘  └──────────────┘                │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Technology Stack

| Layer          | Technology                      |
|---------------|----------------------------------|
| Backend        | Go 1.22+                        |
| Frontend       | Next.js 14 (App Router)         |
| Database       | PostgreSQL 16                   |
| Cache          | Redis 7                         |
| Messaging      | NATS JetStream                  |
| Container      | Docker Compose                  |
| API Style      | REST (gRPC-ready boundaries)    |
| Auth Standards | OAuth2, OIDC, JWT, PKCE         |
| Authz Models   | RBAC, ABAC, PBAC                |
| Migrations     | golang-migrate                  |
| HTTP Router    | chi                             |

---

## Repository Structure

```
unified-trust-platform/
├── backend/
│   ├── cmd/api/                  # Binary entrypoint
│   ├── internal/
│   │   ├── auth/                 # Auth middleware and session
│   │   ├── oidc/                 # OIDC discovery, JWKS
│   │   ├── oauth/                # OAuth2 flows, token engine
│   │   ├── mfa/                  # MFA enrollment and verification
│   │   ├── federation/           # External IdP integration
│   │   ├── delegation/           # Delegated access
│   │   ├── jit/                  # Just-in-time access
│   │   ├── policy/               # Policy engine (RBAC+ABAC+PBAC)
│   │   ├── audit/                # Immutable audit log
│   │   ├── tenant/               # Tenant management
│   │   ├── user/                 # User lifecycle
│   │   ├── client/               # OAuth2 client applications
│   │   ├── database/             # DB connection and repositories
│   │   ├── config/               # Configuration loading
│   │   └── http/                 # Router, middleware, handlers
│   ├── migrations/               # SQL migration files
│   ├── tests/                    # Integration tests
│   ├── go.mod
│   └── Dockerfile
├── frontend/
│   ├── app/                      # Next.js App Router pages
│   ├── components/               # Reusable UI components
│   ├── lib/                      # API client, utilities
│   ├── types/                    # TypeScript types
│   ├── package.json
│   └── Dockerfile
├── deploy/
│   ├── docker-compose.yml
│   └── nginx/
├── docs/
│   ├── architecture.md           # This file
│   ├── security-model.md         # Threat model and controls
│   ├── api-design.md             # API conventions
│   ├── authorization-model.md    # RBAC/ABAC/PBAC design
│   ├── implementation-plan.md    # Phase-by-phase plan
│   └── git-workflow.md           # Branching and commit rules
└── README.md
```

---

## Core Domains

### 1. Identity Domain
- Multi-tenant user directory
- Password lifecycle with policy enforcement
- User status: active, suspended, pending, locked

### 2. Authentication Domain
- OAuth2 Authorization Server
- OpenID Connect Provider
- PKCE-enforced authorization code flow
- JWT access and ID tokens (RS256)
- Refresh token rotation
- Session management and revocation

### 3. MFA Domain
- Pluggable MFA provider interface
- TOTP (RFC 6238) implementation
- OTP challenge/verify cycle
- Per-tenant MFA policy enforcement

### 4. Federation Domain
- External OIDC provider configuration
- Provider metadata discovery
- External user account linking
- Trust status lifecycle

### 5. Authorization Domain
- RBAC: roles, permissions, user-role assignments
- ABAC: subject/resource/environment attributes and conditions
- PBAC: declarative policy documents evaluated at runtime
- Unified evaluation engine returning permit/deny + explanation

### 6. Delegation Domain
- User-to-user access delegation
- Admin delegation grants
- Time-bound and scope-bound constraints
- Delegation audit trail

### 7. JIT Access Domain
- Self-service access request workflow
- Approver assignment and notification
- Temporary privilege grant with automatic expiry
- Emergency access escalation path

### 8. Audit Domain
- Immutable, append-only event store
- Events: login, token, MFA, policy decision, delegation, admin change
- Queryable with filters and export

---

## Security Model

- All tokens signed RS256, verified on every request
- PKCE required for all authorization code flows
- Tenant isolation enforced at every data layer query
- Secrets via environment variables (never committed)
- Rate limiting on authentication endpoints
- Brute-force protection with lockout policy
- Secure HTTP headers (HSTS, CSP, X-Frame-Options)
- Audit every privilege change and access decision

---

## API Design Principles

- Versioned under `/api/v1/`
- Standard error envelope: `{ "error": { "code": "", "message": "", "details": {} } }`
- Request ID propagated through all layers
- Pagination: cursor-based for large collections
- Tenant context via header `X-Tenant-ID` or path prefix
