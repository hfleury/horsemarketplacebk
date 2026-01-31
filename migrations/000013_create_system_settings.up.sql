CREATE TABLE IF NOT EXISTS authentic.system_settings (
    key VARCHAR(255) PRIMARY KEY,
    value VARCHAR(255),
    description TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed default settings
INSERT INTO authentic.system_settings (key, value, description)
VALUES ('product_approval_required', 'true', 'If true, new products require admin approval before going live.');
