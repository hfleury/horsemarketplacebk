CREATE TABLE IF NOT EXISTS authentic.password_resets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES authentic.users(id) ON DELETE CASCADE,
    reset_token TEXT NOT NULL UNIQUE,
    requested_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    is_used BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_resets_user_id ON authentic.password_resets (user_id);
CREATE INDEX idx_password_resets_reset_token ON authentic.password_resets (reset_token);
