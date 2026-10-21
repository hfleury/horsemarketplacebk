ALTER TABLE catalog.payments DROP CONSTRAINT payments_user_id_fkey;
ALTER TABLE catalog.payments
    ADD CONSTRAINT payments_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE;

-- Payments kept after their user was deleted can't satisfy NOT NULL again.
DELETE FROM catalog.payments WHERE user_id IS NULL;
ALTER TABLE catalog.payments ALTER COLUMN user_id SET NOT NULL;

-- Payments block a provider delete (no ON DELETE); their boosts cascade.
DELETE FROM catalog.payments
WHERE provider_id = (SELECT id FROM catalog.payment_providers WHERE code = 'sandbox');
DELETE FROM catalog.payment_providers WHERE code = 'sandbox';
