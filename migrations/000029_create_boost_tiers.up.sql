CREATE TABLE IF NOT EXISTS catalog.boost_tiers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(20) NOT NULL UNIQUE,
    name VARCHAR(50) NOT NULL,
    price_sek DECIMAL(12,2) NOT NULL CHECK (price_sek > 0),
    duration_days INTEGER NOT NULL CHECK (duration_days > 0),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Placeholder prices (SEK incl. 25% moms) until an admin sets researched ones.
INSERT INTO catalog.boost_tiers (code, name, price_sek, duration_days) VALUES
    ('boost', 'Boost', 49.00, 7),
    ('medium', 'Medium', 99.00, 14),
    ('pro', 'Pro', 199.00, 30)
ON CONFLICT (code) DO NOTHING;

-- 000028 declared the tier CHECK inline, so Postgres named it
-- <table>_<column>_check. No IF EXISTS: a wrong name must fail the migration.
ALTER TABLE catalog.product_boosts DROP CONSTRAINT product_boosts_tier_check;

-- No ON DELETE: tiers are deactivated, never deleted, so boosts keep their tier.
ALTER TABLE catalog.product_boosts
    ADD CONSTRAINT product_boosts_tier_fkey FOREIGN KEY (tier) REFERENCES catalog.boost_tiers(code);
