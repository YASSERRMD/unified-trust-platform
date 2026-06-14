# Unified Trust Platform

A production-grade Enterprise Identity and Access Management (IAM) platform providing SSO, MFA, federation, delegation, just-in-time access, and a multi-model authorization engine (RBAC + ABAC + PBAC).

## Stack

| Layer      | Technology                             |
|------------|----------------------------------------|
| Backend    | Go 1.22+ (chi router, zap logging)     |
| Frontend   | Next.js 16 (App Router, TypeScript)    |
| Database   | PostgreSQL 16                          |
| Cache      | Redis 7                                |
| Containers | Docker Compose                         |
| Auth       | OAuth2, OpenID Connect, JWT, PKCE      |
| Authz      | RBAC, ABAC, PBAC                       |

## Quick Start

### Prerequisites
- Go 1.22+
- Node.js 20+
- Docker and Docker Compose
- `make`

### 1. Configure environment

```bash
cp .env.example .env
# Edit .env — set DB_PASSWORD at minimum
```

### 2. Start infrastructure

```bash
make docker-up
```

### 3. Run migrations

```bash
make migrate-up
```

### 4. Start backend

```bash
make backend
```

### 5. Start frontend

```bash
make frontend
```

### Health checks

```
GET http://localhost:8080/health   → { "status": "ok" }
GET http://localhost:8080/ready    → { "db": "ok", "redis": "ok" }
```

## Development Commands

```bash
make help          # list all commands
make test          # run backend tests
make lint          # lint frontend
make fmt           # format backend
make ci            # run all checks
```

## Project Structure

```
unified-trust-platform/
├── backend/          # Go IAM API server
├── frontend/         # Next.js admin portal
├── deploy/           # Docker Compose infrastructure
├── docs/             # Architecture and design documentation
├── Makefile          # Developer commands
└── README.md
```

## Documentation

- [Architecture](docs/architecture.md)
- [Implementation Plan](docs/implementation-plan.md)
- [Security Model](docs/security-model.md)
- [API Design](docs/api-design.md)
- [Authorization Model](docs/authorization-model.md)
- [Git Workflow](docs/git-workflow.md)

## License

MIT — Copyright (c) 2026 Mohamed Yasser
