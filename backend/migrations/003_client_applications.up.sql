CREATE TABLE client_applications (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id          UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    client_id          TEXT NOT NULL UNIQUE,
    client_secret_hash TEXT,
    name               TEXT NOT NULL,
    description        TEXT,
    client_type        TEXT NOT NULL DEFAULT 'confidential'
                       CHECK (client_type IN ('confidential', 'public')),
    grant_types        TEXT[] NOT NULL DEFAULT '{"authorization_code"}',
    scopes             TEXT[] NOT NULL DEFAULT '{"openid","profile","email"}',
    status             TEXT NOT NULL DEFAULT 'active'
                       CHECK (status IN ('active', 'suspended', 'deleted')),
    logo_url           TEXT,
    metadata           JSONB NOT NULL DEFAULT '{}',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE redirect_uris (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id    UUID NOT NULL REFERENCES client_applications (id) ON DELETE CASCADE,
    uri          TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (client_id, uri)
);

CREATE INDEX idx_clients_tenant_id ON client_applications (tenant_id);
CREATE INDEX idx_clients_client_id ON client_applications (client_id);
