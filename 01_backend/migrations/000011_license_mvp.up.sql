CREATE TABLE IF NOT EXISTS licenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    license_no TEXT NOT NULL UNIQUE,
    license_type TEXT NOT NULL DEFAULT 'tenant',
    status TEXT NOT NULL DEFAULT 'inactive',
    subject JSONB NOT NULL DEFAULT '{}'::jsonb,
    limits JSONB NOT NULL DEFAULT '{}'::jsonb,
    public_key_id TEXT,
    signature TEXT,
    issued_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    activated_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    expired_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT chk_licenses_status CHECK (status IN ('inactive', 'active', 'revoked', 'expired')),
    CONSTRAINT chk_licenses_type CHECK (license_type IN ('tenant', 'trial', 'offline'))
);

CREATE INDEX IF NOT EXISTS idx_licenses_tenant_status
ON licenses(tenant_id, status, created_at DESC)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_licenses_tenant_license_no
ON licenses(tenant_id, license_no)
WHERE deleted_at IS NULL;
