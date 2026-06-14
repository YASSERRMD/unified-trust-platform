# Unified Trust Platform — API Design

## Conventions

- Base path: `/api/v1/`
- Format: JSON (`Content-Type: application/json`)
- Auth: Bearer token (`Authorization: Bearer <jwt>`)
- Tenant context: `X-Tenant-ID: <uuid>` header (or sub-path `/t/{tenant}`)
- Request tracing: `X-Request-ID` echoed in response headers
- Pagination: cursor-based (`?cursor=<opaque>&limit=<int>`)

---

## Standard Response Envelope

### Success (2xx)
```json
{
  "data": { ... },
  "meta": {
    "requestId": "uuid",
    "timestamp": "2026-01-01T00:00:00Z"
  }
}
```

### Collection
```json
{
  "data": [ ... ],
  "pagination": {
    "nextCursor": "opaque-cursor",
    "hasMore": true,
    "total": 100
  },
  "meta": { "requestId": "uuid" }
}
```

### Error (4xx / 5xx)
```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Human-readable description",
    "details": { "field": "validation message" }
  },
  "meta": { "requestId": "uuid" }
}
```

---

## Error Codes

| Code | HTTP | Meaning |
|------|------|---------|
| `INVALID_REQUEST` | 400 | Malformed body or params |
| `VALIDATION_FAILED` | 422 | Field-level validation errors |
| `UNAUTHORIZED` | 401 | Missing or invalid token |
| `FORBIDDEN` | 403 | Authenticated but not permitted |
| `NOT_FOUND` | 404 | Resource does not exist |
| `CONFLICT` | 409 | Duplicate resource |
| `RATE_LIMITED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Server error |

---

## API Endpoint Reference

### Meta
```
GET /health              → 200 { status: "ok" }
GET /ready               → 200 { db: "ok", redis: "ok" }
GET /api/v1/meta         → 200 { version, features }
```

### OIDC / OAuth2
```
GET  /.well-known/openid-configuration
GET  /.well-known/jwks.json
GET  /oauth2/authorize?response_type=code&client_id=&redirect_uri=&scope=&state=&code_challenge=&code_challenge_method=S256
POST /oauth2/token       → { access_token, token_type, expires_in, refresh_token, id_token }
POST /oauth2/revoke      → 200
GET  /oauth2/userinfo    → { sub, email, name, ... }
POST /oauth2/logout
```

### Tenants
```
POST   /api/v1/tenants               → 201 Tenant
GET    /api/v1/tenants               → 200 [ ]Tenant
GET    /api/v1/tenants/{id}          → 200 Tenant
PATCH  /api/v1/tenants/{id}          → 200 Tenant
DELETE /api/v1/tenants/{id}          → 204
```

### Users
```
POST   /api/v1/users                 → 201 User
GET    /api/v1/users?q=&status=      → 200 [ ]User
GET    /api/v1/users/{id}            → 200 User
PATCH  /api/v1/users/{id}            → 200 User
DELETE /api/v1/users/{id}            → 204
POST   /api/v1/users/{id}/suspend    → 200
POST   /api/v1/users/{id}/activate   → 200
```

### Client Applications
```
POST   /api/v1/clients               → 201 { client_id, client_secret }
GET    /api/v1/clients               → 200 [ ]Client
GET    /api/v1/clients/{id}          → 200 Client
PATCH  /api/v1/clients/{id}          → 200 Client
DELETE /api/v1/clients/{id}          → 204
```

### Roles & Permissions
```
POST   /api/v1/roles                         → 201 Role
GET    /api/v1/roles                         → 200 [ ]Role
GET    /api/v1/roles/{id}                    → 200 Role
POST   /api/v1/roles/{id}/permissions        → 200
DELETE /api/v1/roles/{id}/permissions/{pid}  → 204
POST   /api/v1/permissions                   → 201 Permission
GET    /api/v1/permissions                   → 200 [ ]Permission
POST   /api/v1/users/{id}/roles              → 200
DELETE /api/v1/users/{id}/roles/{rid}        → 204
```

### Authorization Engine
```
POST /api/v1/authz/evaluate
Body: { subject, action, resource, context }
→ { allowed, decision, reason, matchedPolicies }
```

### Policies
```
POST   /api/v1/policies              → 201 Policy
GET    /api/v1/policies              → 200 [ ]Policy
GET    /api/v1/policies/{id}         → 200 Policy
PUT    /api/v1/policies/{id}         → 200 Policy
DELETE /api/v1/policies/{id}         → 204
```

### MFA
```
GET    /api/v1/mfa/methods           → 200 [ ]MfaMethod
POST   /api/v1/mfa/enroll            → 201 { secret, qrCode }
POST   /api/v1/mfa/challenge         → 200 { challengeId }
POST   /api/v1/mfa/verify            → 200 { token }
DELETE /api/v1/mfa/methods/{id}      → 204
```

### Delegations
```
POST   /api/v1/delegations                → 201 Delegation
GET    /api/v1/delegations                → 200 [ ]Delegation
GET    /api/v1/delegations/{id}           → 200 Delegation
POST   /api/v1/delegations/{id}/approve   → 200
POST   /api/v1/delegations/{id}/revoke    → 200
```

### JIT Access
```
POST   /api/v1/jit/requests               → 201 JitRequest
GET    /api/v1/jit/requests               → 200 [ ]JitRequest
GET    /api/v1/jit/requests/{id}          → 200 JitRequest
POST   /api/v1/jit/requests/{id}/approve  → 200
POST   /api/v1/jit/requests/{id}/reject   → 200
POST   /api/v1/jit/requests/{id}/revoke   → 200
```

### Federation
```
POST   /api/v1/federation/providers           → 201 Provider
GET    /api/v1/federation/providers           → 200 [ ]Provider
GET    /api/v1/federation/providers/{id}      → 200 Provider
POST   /api/v1/federation/providers/{id}/test → 200 { success, latencyMs }
GET    /federation/{provider}/login           → 302 (redirect to IdP)
GET    /federation/{provider}/callback        → 302 (redirect to frontend)
```

### Audit
```
GET /api/v1/audit/events?category=&actor=&from=&to=&cursor= → 200 [ ]AuditEvent
GET /api/v1/audit/events/{id}                               → 200 AuditEvent
GET /api/v1/audit/export?format=json|csv                    → 200 file
```

---

## Pagination

All list endpoints support cursor pagination:

```
GET /api/v1/users?cursor=<opaque>&limit=50
```

Response includes `pagination.nextCursor` when more results exist.

---

## Versioning

Future breaking changes will use `/api/v2/`. The `v1` contract is stable once Phase 3 merges.
