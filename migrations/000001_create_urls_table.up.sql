-- 000001_create_urls_table.up.sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS urls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(32) NOT NULL,
    original_url TEXT NOT NULL,
    clicks BIGINT NOT NULL DEFAULT 0 CHECK (clicks >= 0),
    last_accessed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Partial unique index ensures active codes are unique, while soft-deleted ones do not conflict
CREATE UNIQUE INDEX IF NOT EXISTS idx_urls_code_active ON urls(code) WHERE deleted_at IS NULL;

-- Index for ordering in paginated queries
CREATE INDEX IF NOT EXISTS idx_urls_created_at ON urls(created_at DESC);

-- Index for original URL lookups
CREATE INDEX IF NOT EXISTS idx_urls_original_url ON urls(original_url);
