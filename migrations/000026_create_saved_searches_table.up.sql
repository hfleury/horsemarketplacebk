CREATE TABLE IF NOT EXISTS catalog.saved_searches (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    criteria JSONB NOT NULL,
    location_label TEXT,
    alerts_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    last_notified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_saved_searches_user_id ON catalog.saved_searches(user_id);
