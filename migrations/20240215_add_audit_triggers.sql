-- migrations/20240215_add_audit_triggers.sql
-- ------------------------------------------------------------
-- Add triggers to enforce immutability of the audit_logs table.
-- Any UPDATE or DELETE attempt will be rejected with a clear
-- error message. Additionally, ensure `created_at` is always
-- populated on INSERT.
-- ------------------------------------------------------------

BEGIN;

-- 1. Ensure the audit_logs table exists (defensive check)
--    The table is expected to have at least the following columns:
--      id            UUID PRIMARY KEY
--      tenant_id     UUID NOT NULL
--      user_id       UUID NOT NULL
--      action        TEXT NOT NULL
--      resource_type TEXT NOT NULL
--      resource_id   UUID NOT NULL
--      details       JSONB
--      created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
--    If the schema differs, adjust the trigger function accordingly.

-- 2. Create (or replace) a function that blocks UPDATE and DELETE.
CREATE OR REPLACE FUNCTION public.audit_logs_immutable()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    RAISE EXCEPTION 'Audit logs are immutable and cannot be %.', TG_OP
        USING ERRCODE = '55006';  -- read_only_sql_transaction
END;
$$;

-- 3. Attach BEFORE UPDATE trigger to prevent modifications.
DROP TRIGGER IF EXISTS trg_audit_logs_prevent_update ON public.audit_logs;
CREATE TRIGGER trg_audit_logs_prevent_update
BEFORE UPDATE ON public.audit_logs
FOR EACH ROW
EXECUTE FUNCTION public.audit_logs_immutable();

-- 4. Attach BEFORE DELETE trigger to prevent deletions.
DROP TRIGGER IF EXISTS trg_audit_logs_prevent_delete ON public.audit_logs;
CREATE TRIGGER trg_audit_logs_prevent_delete
BEFORE DELETE ON public.audit_logs
FOR EACH ROW
EXECUTE FUNCTION public.audit_logs_immutable();

-- 5. (Optional) Ensure created_at is always set on INSERT.
--    This is a safety net in case callers omit the column.
CREATE OR REPLACE FUNCTION public.audit_logs_set_timestamp()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
BEGIN
    IF NEW.created_at IS NULL THEN
        NEW.created_at := now();
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_audit_logs_set_timestamp ON public.audit_logs;
CREATE TRIGGER trg_audit_logs_set_timestamp
BEFORE INSERT ON public.audit_logs
FOR EACH ROW
EXECUTE FUNCTION public.audit_logs_set_timestamp();

COMMIT;