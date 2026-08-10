CREATE SCHEMA IF NOT EXISTS authentic AUTHORIZATION horsemktuser;

ALTER TABLE auth.users SET SCHEMA authentic;
ALTER TABLE auth.user_sessions SET SCHEMA authentic;
ALTER TABLE auth.password_resets SET SCHEMA authentic;
ALTER TABLE auth.multi_factor_auth SET SCHEMA authentic;
ALTER TABLE auth.email_verifications SET SCHEMA authentic;

ALTER TABLE catalog.categories SET SCHEMA authentic;
ALTER TABLE catalog.products SET SCHEMA authentic;
ALTER TABLE catalog.product_horses SET SCHEMA authentic;
ALTER TABLE catalog.product_vehicles SET SCHEMA authentic;
ALTER TABLE catalog.product_equipment SET SCHEMA authentic;
ALTER TABLE catalog.product_media SET SCHEMA authentic;
ALTER TABLE catalog.horse_attribute_options SET SCHEMA authentic;
ALTER TYPE catalog.product_status SET SCHEMA public;
ALTER TYPE catalog.product_type SET SCHEMA public;

ALTER TABLE media.media SET SCHEMA authentic;

ALTER TABLE system.system_settings SET SCHEMA authentic;

DROP SCHEMA auth;
DROP SCHEMA catalog;
DROP SCHEMA media;
DROP SCHEMA system;
