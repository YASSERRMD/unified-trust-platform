CREATE TABLE delegations (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    delegator_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    delegate_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    scope_type      TEXT NOT NULL DEFAULT 'role' CHECK (scope_type IN ('role', 'permission', 'full')),
    granted_roles   UUID[],
    granted_permissions UUID[],
    reason          TEXT,
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'active', 'revoked', 'expired')),
    approved_by     UUID REFERENCES users (id) ON DELETE SET NULL,
    approved_at     TIMESTAMPTZ,
    starts_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ,
    revoked_by      UUID REFERENCES users (id) ON DELETE SET NULL,
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_delegations_tenant_id ON delegations (tenant_id);
CREATE INDEX idx_delegations_delegator_id ON delegations (delegator_id);
CREATE INDEX idx_delegations_delegate_id ON delegations (delegate_id);
CREATE INDEX idx_delegations_status ON delegations (status);
