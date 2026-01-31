CREATE TYPE product_status AS ENUM ('draft', 'published', 'pending_approval', 'sold', 'archived', 'deleted');
CREATE TYPE product_type AS ENUM ('horse', 'vehicle', 'equipment', 'service', 'property');

CREATE TABLE IF NOT EXISTS authentic.products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES authentic.users(id) ON DELETE CASCADE,
    category_id UUID REFERENCES authentic.categories(id) ON DELETE SET NULL,
    type product_type NOT NULL,
    status product_status NOT NULL DEFAULT 'draft',
    title VARCHAR(255) NOT NULL,
    price_sek DECIMAL(12, 2),
    description TEXT,
    city VARCHAR(100),
    area VARCHAR(100), -- Area/Region
    transaction_type VARCHAR(50), -- Sale, Rent, Etc.
    views_count INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_products_user_id ON authentic.products(user_id);
CREATE INDEX idx_products_status ON authentic.products(status);
CREATE INDEX idx_products_category_id ON authentic.products(category_id);
CREATE INDEX idx_products_type ON authentic.products(type);

-- Product Horses
CREATE TABLE IF NOT EXISTS authentic.product_horses (
    product_id UUID PRIMARY KEY REFERENCES authentic.products(id) ON DELETE CASCADE,
    name VARCHAR(255),
    age INT, -- Derived from Year of Birth or just Age? User said Year of Birthday
    year_of_birth INT,
    gender VARCHAR(50), -- Mare, Stallion, Gelding
    height INT, -- cm
    breed VARCHAR(100),
    color VARCHAR(50),
    dressage_level VARCHAR(50),
    jump_level VARCHAR(50), 
    orientation VARCHAR(100), -- Discipline?
    pedigree JSONB -- Store nested family tree
);

-- Product Vehicles (Trailers, Trucks)
CREATE TABLE IF NOT EXISTS authentic.product_vehicles (
    product_id UUID PRIMARY KEY REFERENCES authentic.products(id) ON DELETE CASCADE,
    make VARCHAR(100),
    model VARCHAR(100),
    year INT,
    load_weight INT,
    total_weight INT,
    condition VARCHAR(50)
);

-- Product Equipment (Saddles, blankets, etc AND Equestrian equipment)
CREATE TABLE IF NOT EXISTS authentic.product_equipment (
    product_id UUID PRIMARY KEY REFERENCES authentic.products(id) ON DELETE CASCADE,
    make VARCHAR(100),
    model VARCHAR(100),
    size VARCHAR(50),
    condition VARCHAR(50),
    sub_type VARCHAR(50), -- Dressage, Jumping (for saddles)
    boom_width VARCHAR(50) -- specific to saddles
);

-- Link products to media files (Videos and Images)
CREATE TABLE IF NOT EXISTS authentic.product_media (
    product_id UUID NOT NULL REFERENCES authentic.products(id) ON DELETE CASCADE,
    media_id UUID NOT NULL REFERENCES authentic.media(id) ON DELETE CASCADE,
    "order" INT DEFAULT 0,
    is_primary BOOLEAN DEFAULT FALSE,
    PRIMARY KEY (product_id, media_id)
);
