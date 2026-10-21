-- Test-only provider. Whether it can be used is decided by config, not by this row.
INSERT INTO catalog.payment_providers (code, name)
VALUES ('sandbox', 'Sandbox (test only)')
ON CONFLICT (code) DO NOTHING;

-- Bokföringslagen requires keeping payment records for 7 years, so deleting a
-- user must not delete their payments.
ALTER TABLE catalog.payments ALTER COLUMN user_id DROP NOT NULL;

-- 000028 declared the FK inline, so Postgres named it <table>_<column>_fkey.
-- No IF EXISTS: a wrong name must fail the migration.
ALTER TABLE catalog.payments DROP CONSTRAINT payments_user_id_fkey;
ALTER TABLE catalog.payments
    ADD CONSTRAINT payments_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE SET NULL;
