# Unified Trust Platform — Security Model

## Threat Model

### Assets
- User credentials (passwords, MFA secrets, tokens)
- Tenant data (users, roles, policies, audit logs)
- OAuth2 client secrets
- JWT signing keys

### Threat Actors
- **External attacker**: no credentials, targets public endpoints
- **Insider threat**: valid user attempting privilege escalation
- **Compromised client**: OAuth2 client with stolen secret
- **Rogue tenant admin**: attempting cross-tenant data access

---

## Authentication Controls

| Control | Implementation |
|---------|---------------|
| Password storage | bcrypt (cost 12) |
| Password policy | Min 12 chars, uppercase, lowercase, digit, symbol |
| Brute-force protection | 5 failures → 15-min lockout, exponential backoff |
| MFA | TOTP (RFC 6238), OTP via email/SMS |
| Session tokens | Random 32-byte, Redis-backed, TTL enforced |
| JWT algorithm | RS256 only, key rotation supported |
| PKCE | S256 required for all auth code flows |

---

## Authorization Controls

| Control | Implementation |
|---------|---------------|
| Tenant isolation | Every DB query scoped to tenant_id |
| Token verification | Signature + expiry + audience + tenant claim |
| RBAC | Role-permission matrix evaluated on every request |
| ABAC | Attribute conditions evaluated at policy engine |
| PBAC | Declarative policy documents, deny-by-default |
| Admin actions | Require elevated session + audit log |

---

## Transport Controls

| Control | Value |
|---------|-------|
| TLS | Required in production (nginx terminates) |
| HSTS | `max-age=31536000; includeSubDomains` |
| Content-Security-Policy | Strict; no unsafe-inline |
| X-Frame-Options | DENY |
| X-Content-Type-Options | nosniff |
| Referrer-Policy | strict-origin-when-cross-origin |
| CORS | Allowlist-based, credentials=true only for trusted origins |

---

## Secret Management

- All secrets via environment variables
- No secrets committed to git
- Signing keys: generated at startup if not provided, stored in env/secrets manager
- Client secrets: SHA-256 hashed before storage, shown once on creation
- Database passwords: env var only
- Redis auth: env var only

---

## Rate Limiting

| Endpoint | Limit |
|----------|-------|
| `POST /oauth2/token` | 20 req/min per IP |
| `GET /oauth2/authorize` | 60 req/min per IP |
| `POST /api/v1/users` | 30 req/min per tenant |
| `POST /api/v1/mfa/verify` | 10 req/min per user |
| All other API | 300 req/min per IP |

---

## Audit Events

Every security-relevant action produces an immutable audit event:

| Category | Events |
|----------|--------|
| Authentication | login_success, login_failure, logout, token_issued, token_revoked |
| MFA | mfa_enrolled, mfa_verified, mfa_failed, mfa_disabled |
| Authorization | policy_permit, policy_deny, role_assigned, role_revoked |
| Delegation | delegation_created, delegation_approved, delegation_revoked |
| JIT | jit_requested, jit_approved, jit_rejected, jit_expired |
| Admin | user_created, user_suspended, tenant_created, client_registered |
| Federation | idp_linked, idp_login, account_linked |

---

## Known Production Gaps (to address post-MVP)

- Hardware security module (HSM) for signing key storage
- Anomaly detection on login patterns
- IP geolocation-based risk scoring
- Certificate pinning for mobile clients
- FIDO2/WebAuthn support
- Secrets rotation automation (Vault integration)
