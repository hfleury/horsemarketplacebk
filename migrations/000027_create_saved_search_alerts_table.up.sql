CREATE TABLE IF NOT EXISTS catalog.saved_search_alerts (
    saved_search_id UUID NOT NULL REFERENCES catalog.saved_searches(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES catalog.products(id) ON DELETE CASCADE,
    sent_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (saved_search_id, product_id)
);
CREATE INDEX idx_saved_search_alerts_product_id ON catalog.saved_search_alerts(product_id);
