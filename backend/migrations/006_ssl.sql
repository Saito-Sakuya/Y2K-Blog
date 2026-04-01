-- SSL/TLS settings
INSERT INTO settings (key, value) VALUES ('frontend_domain', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO settings (key, value) VALUES ('admin_domain', '') ON CONFLICT (key) DO NOTHING;

INSERT INTO settings (key, value) VALUES ('frontend_ssl_enabled', 'false') ON CONFLICT (key) DO NOTHING;
INSERT INTO settings (key, value) VALUES ('frontend_ssl_cert_pem', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO settings (key, value) VALUES ('frontend_ssl_key_pem', '') ON CONFLICT (key) DO NOTHING;

INSERT INTO settings (key, value) VALUES ('admin_ssl_enabled', 'false') ON CONFLICT (key) DO NOTHING;
INSERT INTO settings (key, value) VALUES ('admin_ssl_cert_pem', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO settings (key, value) VALUES ('admin_ssl_key_pem', '') ON CONFLICT (key) DO NOTHING;
