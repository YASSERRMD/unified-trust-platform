CREATE TABLE mfa_methods (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    tenant_id   UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    method_type TEXT NOT NULL CHECK (method_type IN ('totp', 'email_otp', 'sms_otp', 'webauthn')),
    secret_hash TEXT,
    name        TEXT NOT NULL DEFAULT 'default',
    is_primary  BOOLEAN NOT NULL DEFAULT FALSE,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    metadata    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, method_type, name)
);

CREATE TABLE mfa_challenges (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    method_id   UUID REFERENCES mfa_methods (id) ON DELETE CASCADE,
    method_type TEXT NOT NULL,
    otp_hash    TEXT,
    attempts    INT NOT NULL DEFAULT 0,
    is_used     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_mfa_methods_user_id ON mfa_methods (user_id);
CREATE INDEX idx_mfa_challenges_user_id ON mfa_challenges (user_id);
