-- migrations/20240101_create_tables.sql
-- ------------------------------------------------------------
-- This migration creates the core tables for FinSight AI:
--   * documents      – metadata for uploaded financial documents
--   * audit_logs    – immutable audit trail of system actions
--   * webhooks       – subscription definitions for outbound events
-- It also defines supporting enum types and indexes required for
-- production‑grade query performance and data integrity.
-- ------------------------------------------------------------

-- Enable the pgcrypto extension for UUID generation (if not already present)
CREATE EXTENSION IF NOT EXISTS pgcrypto;

--------------------------------------------------------------------
-- ENUM TYPES
--------------------------------------------------------------------
-- Document processing status
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'document_status') THEN
        CREATE TYPE document_status AS ENUM (
            'uploaded',   -- file received, waiting for queue
            'queued',     -- task enqueued
            'processing',-- OCR/LLM in progress
            'completed', -- extraction succeeded
            'failed',     -- processing error
            'rejected'   -- manual rejection by user
        );
    END IF;
END$$;

--------------------------------------------------------------------
-- TABLE: documents
--------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS documents (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL,
    uploaded_by       UUID NOT NULL,               -- reference to users table
    storage_key       TEXT NOT NULL,               -- object key in MinIO
    filename          TEXT NOT NULL,
    content_type      TEXT NOT NULL,
    file_size_bytes   BIGINT NOT NULL CHECK (file_size_bytes >= 0),
    status            document_status NOT NULL DEFAULT 'uploaded',
    extracted_data    JSONB,                       -- structured result from LLM
    raw_ocr_text      TEXT,                        -- optional raw OCR output
    metadata          JSONB DEFAULT '{}'::jsonb,   -- any additional user‑supplied data
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at     TIMESTAMPTZ,                 -- set when status becomes completed/failed
    CONSTRAINT fk_documents_tenant
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_documents_user
        FOREIGN KEY (uploaded_by) REFERENCES users(id) ON DELETE RESTRICT
);

-- Indexes for fast tenant‑scoped queries and status filtering
CREATE INDEX IF NOT EXISTS idx_documents_tenant_status
    ON documents (tenant_id, status);

CREATE INDEX IF NOT EXISTS idx_documents_tenant_created
    ON documents (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_documents_tenant_processed
    ON documents (tenant_id, processed_at DESC);

--------------------------------------------------------------------
-- TABLE: audit_logs
--------------------------------------------------------------------
-- Immutable audit log; never updated or deleted.
CREATE TABLE IF NOT EXISTS audit_logs (
    id                BIGSERIAL PRIMARY KEY,
    tenant_id         UUID NOT NULL,
    user_id           UUID,                        -- actor; may be NULL for system actions
    action            TEXT NOT NULL,                -- e.g., "document.uploaded"
    table_name        TEXT NOT NULL,                -- affected table
    record_id         TEXT NOT NULL,                -- primary key of the affected record (as text for flexibility)
    before_data       JSONB,                        -- snapshot before change
    after_data        JSONB,                        -- snapshot after change
    ip_address        INET,                         -- source IP
    user_agent        TEXT,                         -- client user‑agent string
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_audit_tenant
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_audit_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

-- Indexes to support tenant‑wide audit queries
CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_created
    ON audit_logs (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user
    ON audit_logs (user_id);

--------------------------------------------------------------------
-- TABLE: webhooks
--------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS webhooks (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL,
    created_by        UUID NOT NULL,               -- owner of the subscription
    url               TEXT NOT NULL,
    secret            TEXT NOT NULL,               -- HMAC secret (stored encrypted at app level)
    events            TEXT[] NOT NULL,              -- list of event names the subscriber wants
    enabled           BOOLEAN NOT NULL DEFAULT TRUE,
    failure_count     INTEGER NOT NULL DEFAULT 0,
    last_failure_at   TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_webhook_tenant
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_webhook_user
        FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT uq_webhook_tenant_url
        UNIQUE (tenant_id, url)
);

-- Indexes for efficient dispatching and health monitoring
CREATE INDEX IF NOT EXISTS idx_webhooks_tenant_enabled
    ON webhooks (tenant_id, enabled);

CREATE INDEX IF NOT EXISTS idx_webhooks_failure
    ON webhooks (tenant_id, failure_count DESC, last_failure_at);

--------------------------------------------------------------------
-- TRIGGERS
--------------------------------------------------------------------
-- Auto‑update `updated_at` on row modification for documents and webhooks
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_documents_updated
BEFORE UPDATE ON documents
FOR EACH ROW
EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER trg_webhooks_updated
BEFORE UPDATE ON webhooks
FOR EACH ROW
EXECUTE FUNCTION update_timestamp();

-- End of migration
--------------------------------------------------------------------