ALTER TABLE catalog.product_boosts DROP CONSTRAINT IF EXISTS product_boosts_tier_fkey;

-- Fails if a boost uses an admin-created tier; that boost's payment history
-- must be resolved by hand before rolling back.
ALTER TABLE catalog.product_boosts
    ADD CONSTRAINT product_boosts_tier_check CHECK (tier IN ('boost', 'medium', 'pro'));

DROP TABLE IF EXISTS catalog.boost_tiers;
