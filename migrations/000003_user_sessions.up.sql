CREATE TABLE authentic.user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES authentic.users(id) ON DELETE CASCADE,
    session_token TEXT NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_activity TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_user_sessions_session_token ON authentic.user_sessions (session_token);

ALTER TABLE authentic.user_sessions
    ADD CONSTRAINT fk_user_sessions_user_id FOREIGN KEY (user_id)
    REFERENCES authentic.users(id) ON DELETE CASCADE;
