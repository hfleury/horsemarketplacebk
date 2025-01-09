CREATE TABLE IF NOT EXISTS authentic.multi_factor_auth (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES authentic.users(id) ON DELETE CASCADE,
    mfa_type VARCHAR(50) NOT NULL, -- e.g., 'SMS', 'TOTP', etc.
    mfa_secret TEXT NOT NULL,      -- The secret key used for generating OTP or similar
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create an index for faster lookups
CREATE INDEX idx_multi_factor_auth_user_id ON authentic.multi_factor_auth (user_id);
CREATE INDEX idx_multi_factor_auth_mfa_type ON authentic.multi_factor_auth (mfa_type);
