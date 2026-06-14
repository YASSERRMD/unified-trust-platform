CREATE TABLE sessions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    client_id       UUID REFERENCES client_applications (id) ON DELETE SET NULL,
    token_hash      TEXT NOT NULL UNIQUE,
    ip_address      INET,
    user_agent      TEXT,
    scopes          TEXT[] NOT NULL DEFAULT '{}',
    auth_methods    TEXT[] NOT NULL DEFAULT '{}',
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    last_active_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tokens (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id      UUID REFERENCES sessions (id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    client_id       UUID REFERENCES client_applications (id) ON DELETE SET NULL,
    token_type      TEXT NOT NULL CHECK (token_type IN ('access', 'refresh', 'authorization_code', 'id')),
    token_hash      TEXT NOT NULL UNIQUE,
    jti             TEXT UNIQUE,
    scopes          TEXT[] NOT NULL DEFAULT '{}',
    is_revoked      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ
);

CREATE TABLE authorization_codes (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code_hash           TEXT NOT NULL UNIQUE,
    client_id           UUID NOT NULL REFERENCES client_applications (id) ON DELETE CASCADE,
    user_id             UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    redirect_uri        TEXT NOT NULL,
    scopes              TEXT[] NOT NULL DEFAULT '{}',
    code_challenge      TEXT,
    code_challenge_method TEXT,
    nonce               TEXT,
    is_used             BOOLEAN NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_sessions_user_id ON sessions (user_id);
CREATE INDEX idx_sessions_tenant_id ON sessions (tenant_id);
CREATE INDEX idx_sessions_token_hash ON sessions (token_hash);
CREATE INDEX idx_tokens_user_id ON tokens (user_id);
CREATE INDEX idx_tokens_jti ON tokens (jti);
CREATE INDEX idx_tokens_token_type ON tokens (token_type);
CREATE INDEX idx_auth_codes_code_hash ON authorization_codes (code_hash);
