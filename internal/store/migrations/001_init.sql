CREATE TABLE IF NOT EXISTS consent_records (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id TEXT NOT NULL,
    subject_id TEXT NOT NULL,
    purpose TEXT NOT NULL,
    data_category TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('granted', 'revoked')),
    source TEXT NOT NULL DEFAULT 'api',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT consent_records_scope_unique UNIQUE (tenant_id, subject_id, purpose, data_category)
);
CREATE INDEX IF NOT EXISTS consent_records_lookup_idx
    ON consent_records (tenant_id, subject_id, purpose, data_category, status);
CREATE TABLE IF NOT EXISTS audit_events (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS audit_events_tenant_created_idx
    ON audit_events (tenant_id, created_at DESC);
