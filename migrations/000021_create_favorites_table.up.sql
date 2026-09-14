CREATE TABLE IF NOT EXISTS catalog.favorites (
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES catalog.products(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, product_id)
);
CREATE INDEX idx_favorites_product_id ON catalog.favorites(product_id);
