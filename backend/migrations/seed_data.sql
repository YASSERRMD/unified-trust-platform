-- Seed data for development and testing

-- Default tenant
INSERT INTO tenants (id, name, slug, status)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'System Tenant',
    'system',
    'active'
) ON CONFLICT (slug) DO NOTHING;

-- System roles
INSERT INTO roles (id, tenant_id, name, description, is_system)
VALUES
    ('00000000-0000-0000-0001-000000000001', '00000000-0000-0000-0000-000000000001',
     'super-admin', 'Full platform access', true),
    ('00000000-0000-0000-0001-000000000002', '00000000-0000-0000-0000-000000000001',
     'tenant-admin', 'Full tenant administration', true),
    ('00000000-0000-0000-0001-000000000003', '00000000-0000-0000-0000-000000000001',
     'viewer', 'Read-only access', true)
ON CONFLICT (tenant_id, name) DO NOTHING;

-- System permissions
INSERT INTO permissions (id, tenant_id, name, resource, action, is_system)
VALUES
    ('00000000-0000-0000-0002-000000000001', '00000000-0000-0000-0000-000000000001',
     'user:read', 'user', 'read', true),
    ('00000000-0000-0000-0002-000000000002', '00000000-0000-0000-0000-000000000001',
     'user:write', 'user', 'write', true),
    ('00000000-0000-0000-0002-000000000003', '00000000-0000-0000-0000-000000000001',
     'role:read', 'role', 'read', true),
    ('00000000-0000-0000-0002-000000000004', '00000000-0000-0000-0000-000000000001',
     'role:write', 'role', 'write', true),
    ('00000000-0000-0000-0002-000000000005', '00000000-0000-0000-0000-000000000001',
     'policy:read', 'policy', 'read', true),
    ('00000000-0000-0000-0002-000000000006', '00000000-0000-0000-0000-000000000001',
     'policy:write', 'policy', 'write', true),
    ('00000000-0000-0000-0002-000000000007', '00000000-0000-0000-0000-000000000001',
     'audit:read', 'audit', 'read', true),
    ('00000000-0000-0000-0002-000000000008', '00000000-0000-0000-0000-000000000001',
     'tenant:read', 'tenant', 'read', true),
    ('00000000-0000-0000-0002-000000000009', '00000000-0000-0000-0000-000000000001',
     'tenant:write', 'tenant', 'write', true)
ON CONFLICT (tenant_id, resource, action) DO NOTHING;
