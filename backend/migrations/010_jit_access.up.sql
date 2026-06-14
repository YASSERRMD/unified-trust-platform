CREATE TABLE jit_requests (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    requester_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    target_role_id  UUID REFERENCES roles (id) ON DELETE SET NULL,
    target_permission_id UUID REFERENCES permissions (id) ON DELETE SET NULL,
    access_type     TEXT NOT NULL DEFAULT 'standard'
                    CHECK (access_type IN ('standard', 'emergency')),
    reason          TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'approved', 'rejected', 'revoked', 'expired')),
    requested_duration INTERVAL,
    approver_id     UUID REFERENCES users (id) ON DELETE SET NULL,
    approved_at     TIMESTAMPTZ,
    rejected_at     TIMESTAMPTZ,
    rejection_reason TEXT,
    revoked_by      UUID REFERENCES users (id) ON DELETE SET NULL,
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE jit_grants (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_id      UUID NOT NULL REFERENCES jit_requests (id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    role_id         UUID REFERENCES roles (id) ON DELETE SET NULL,
    permission_id   UUID REFERENCES permissions (id) ON DELETE SET NULL,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    granted_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ
);

CREATE INDEX idx_jit_requests_tenant_id ON jit_requests (tenant_id);
CREATE INDEX idx_jit_requests_requester_id ON jit_requests (requester_id);
CREATE INDEX idx_jit_requests_status ON jit_requests (status);
CREATE INDEX idx_jit_grants_user_id ON jit_grants (user_id);
CREATE INDEX idx_jit_grants_expires_at ON jit_grants (expires_at) WHERE is_active = TRUE;
