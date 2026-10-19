CREATE TABLE IF NOT EXISTS catalog.payment_providers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(30) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS catalog.payments (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    provider_id UUID NOT NULL REFERENCES catalog.payment_providers(id),
    provider_reference VARCHAR(255),
    amount_sek DECIMAL(12,2) NOT NULL CHECK (amount_sek >= 0),
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'paid', 'failed', 'refunded')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (provider_id, provider_reference)
);
CREATE INDEX idx_payments_user_id ON catalog.payments(user_id);
CREATE INDEX idx_payments_status ON catalog.payments(status);

CREATE TABLE IF NOT EXISTS catalog.product_boosts (
    id UUID PRIMARY KEY,
    product_id UUID NOT NULL REFERENCES catalog.products(id) ON DELETE CASCADE,
    payment_id UUID NOT NULL UNIQUE REFERENCES catalog.payments(id) ON DELETE CASCADE,
    tier VARCHAR(20) NOT NULL CHECK (tier IN ('boost', 'medium', 'pro')),
    starts_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (expires_at > starts_at)
);
CREATE INDEX idx_product_boosts_product_id ON catalog.product_boosts(product_id);
