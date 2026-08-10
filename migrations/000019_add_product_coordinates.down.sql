DROP INDEX IF EXISTS catalog.idx_products_location;
ALTER TABLE catalog.products DROP COLUMN longitude;
ALTER TABLE catalog.products DROP COLUMN latitude;
