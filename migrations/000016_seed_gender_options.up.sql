INSERT INTO authentic.horse_attribute_options (id, type, value) VALUES
(uuid_generate_v4(), 'gender', 'Mare') ON CONFLICT DO NOTHING;
INSERT INTO authentic.horse_attribute_options (id, type, value) VALUES
(uuid_generate_v4(), 'gender', 'Stallion') ON CONFLICT DO NOTHING;
INSERT INTO authentic.horse_attribute_options (id, type, value) VALUES
(uuid_generate_v4(), 'gender', 'Gelding') ON CONFLICT DO NOTHING;
