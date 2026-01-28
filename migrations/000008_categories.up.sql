CREATE TABLE IF NOT EXISTS authentic.categories (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    picture_url TEXT,
    parent_id UUID REFERENCES authentic.categories(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_categories_parent_id ON authentic.categories(parent_id);
