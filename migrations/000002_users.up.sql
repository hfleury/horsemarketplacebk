CREATE TABLE IF NOT EXISTS authentic.users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(255) UNIQUE NOT NULL,          
    email VARCHAR(255) UNIQUE NOT NULL,             
    password_hash TEXT NOT NULL,                    
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    last_login TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create an index on username and email for faster lookups
CREATE INDEX idx_users_username ON authentic.users (username);
CREATE INDEX idx_users_email ON authentic.users (email);