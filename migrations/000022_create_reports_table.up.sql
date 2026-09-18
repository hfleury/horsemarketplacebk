CREATE TABLE IF NOT EXISTS catalog.reports (
    id UUID PRIMARY KEY,
    product_id UUID NOT NULL REFERENCES catalog.products(id) ON DELETE CASCADE,
    reporter_user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    reason VARCHAR(20) NOT NULL CHECK (reason IN ('spam', 'fraud', 'inappropriate', 'duplicate', 'other')),
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'reviewed', 'dismissed')),
    reviewed_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX idx_reports_product_id ON catalog.reports(product_id);
CREATE INDEX idx_reports_status ON catalog.reports(status);
