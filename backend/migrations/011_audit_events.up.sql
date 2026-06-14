CREATE TABLE audit_events (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   UUID REFERENCES tenants (id) ON DELETE SET NULL,
    actor_id    UUID REFERENCES users (id) ON DELETE SET NULL,
    actor_email TEXT,
    category    TEXT NOT NULL CHECK (category IN (
                    'authentication', 'authorization', 'mfa',
                    'delegation', 'jit', 'federation', 'admin', 'token'
                )),
    event_type  TEXT NOT NULL,
    resource_type TEXT,
    resource_id TEXT,
    outcome     TEXT NOT NULL CHECK (outcome IN ('success', 'failure', 'error')),
    ip_address  INET,
    user_agent  TEXT,
    metadata    JSONB NOT NULL DEFAULT '{}',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_events_tenant_id ON audit_events (tenant_id);
CREATE INDEX idx_audit_events_actor_id ON audit_events (actor_id);
CREATE INDEX idx_audit_events_category ON audit_events (category);
CREATE INDEX idx_audit_events_occurred_at ON audit_events (occurred_at DESC);
CREATE INDEX idx_audit_events_event_type ON audit_events (event_type);

-- Prevent updates and deletes to keep audit log immutable
CREATE RULE no_update_audit AS ON UPDATE TO audit_events DO INSTEAD NOTHING;
CREATE RULE no_delete_audit AS ON DELETE TO audit_events DO INSTEAD NOTHING;
