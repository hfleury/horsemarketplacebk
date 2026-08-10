CREATE SCHEMA IF NOT EXISTS auth AUTHORIZATION horsemktuser;
CREATE SCHEMA IF NOT EXISTS catalog AUTHORIZATION horsemktuser;
CREATE SCHEMA IF NOT EXISTS media AUTHORIZATION horsemktuser;
CREATE SCHEMA IF NOT EXISTS system AUTHORIZATION horsemktuser;

-- auth domain
ALTER TABLE authentic.users SET SCHEMA auth;
ALTER TABLE authentic.user_sessions SET SCHEMA auth;
ALTER TABLE authentic.password_resets SET SCHEMA auth;
ALTER TABLE authentic.multi_factor_auth SET SCHEMA auth;
ALTER TABLE authentic.email_verifications SET SCHEMA auth;

-- catalog domain
ALTER TABLE authentic.categories SET SCHEMA catalog;
ALTER TABLE authentic.products SET SCHEMA catalog;
ALTER TABLE authentic.product_horses SET SCHEMA catalog;
ALTER TABLE authentic.product_vehicles SET SCHEMA catalog;
ALTER TABLE authentic.product_equipment SET SCHEMA catalog;
ALTER TABLE authentic.product_media SET SCHEMA catalog;
ALTER TABLE authentic.horse_attribute_options SET SCHEMA catalog;
ALTER TYPE public.product_status SET SCHEMA catalog;
ALTER TYPE public.product_type SET SCHEMA catalog;

-- media domain
ALTER TABLE authentic.media SET SCHEMA media;

-- system domain
ALTER TABLE authentic.system_settings SET SCHEMA system;

DROP SCHEMA authentic;
