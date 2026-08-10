ALTER TABLE catalog.products ADD COLUMN latitude DOUBLE PRECISION;
ALTER TABLE catalog.products ADD COLUMN longitude DOUBLE PRECISION;
CREATE INDEX idx_products_location ON catalog.products
  USING GIST ((ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)::geography))
  WHERE latitude IS NOT NULL AND longitude IS NOT NULL;
