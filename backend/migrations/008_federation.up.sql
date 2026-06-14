CREATE TABLE federation_providers (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL,
    provider_type   TEXT NOT NULL CHECK (provider_type IN ('oidc', 'saml', 'ldap')),
    issuer_url      TEXT,
    client_id       TEXT,
    client_secret_hash TEXT,
    scopes          TEXT[] NOT NULL DEFAULT '{"openid","profile","email"}',
    metadata_url    TEXT,
    metadata        JSONB NOT NULL DEFAULT '{}',
    trust_status    TEXT NOT NULL DEFAULT 'pending'
                    CHECK (trust_status IN ('pending', 'active', 'suspended', 'deleted')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, slug)
);

CREATE TABLE linked_identities (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider_id     UUID NOT NULL REFERENCES federation_providers (id) ON DELETE CASCADE,
    external_id     TEXT NOT NULL,
    external_email  TEXT,
    access_token    TEXT,
    refresh_token   TEXT,
    token_expires_at TIMESTAMPTZ,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider_id, external_id)
);

CREATE INDEX idx_federation_providers_tenant_id ON federation_providers (tenant_id);
CREATE INDEX idx_linked_identities_user_id ON linked_identities (user_id);
