CREATE TABLE policies (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id    UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    description  TEXT,
    effect       TEXT NOT NULL DEFAULT 'permit' CHECK (effect IN ('permit', 'deny')),
    priority     INT NOT NULL DEFAULT 100,
    subjects     JSONB NOT NULL DEFAULT '{}',
    actions      TEXT[] NOT NULL DEFAULT '{}',
    resources    JSONB NOT NULL DEFAULT '{}',
    conditions   JSONB NOT NULL DEFAULT '[]',
    time_constraints JSONB,
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_by   UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

CREATE TABLE policy_rules (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_id   UUID NOT NULL REFERENCES policies (id) ON DELETE CASCADE,
    attribute   TEXT NOT NULL,
    operator    TEXT NOT NULL,
    value       JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_policies_tenant_id ON policies (tenant_id);
CREATE INDEX idx_policies_is_active ON policies (is_active);
CREATE INDEX idx_policy_rules_policy_id ON policy_rules (policy_id);
