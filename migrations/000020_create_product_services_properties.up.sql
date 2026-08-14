CREATE TABLE IF NOT EXISTS catalog.product_services (
    product_id UUID PRIMARY KEY REFERENCES catalog.products(id) ON DELETE CASCADE,
    service_type VARCHAR(50),   -- e.g. boarding, training, co_rider, breeding
    availability VARCHAR(100)
);

CREATE TABLE IF NOT EXISTS catalog.product_properties (
    product_id UUID PRIMARY KEY REFERENCES catalog.products(id) ON DELETE CASCADE,
    size_m2 INT,
    room_count INT,
    has_stable BOOLEAN DEFAULT FALSE
);
