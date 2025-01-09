CREATE TABLE IF NOT EXISTS authentic.email_verifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES authentic.users(id) ON DELETE CASCADE,
    verification_token TEXT NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    requested_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create an index for faster lookups
CREATE INDEX idx_email_verifications_user_id ON authentic.email_verifications (user_id);
CREATE INDEX idx_email_verifications_token ON authentic.email_verifications (verification_token);
