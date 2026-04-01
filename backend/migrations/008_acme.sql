-- SSL mode: "off", "manual" (PEM upload), "auto" (Let's Encrypt)
INSERT INTO settings (key, value) VALUES ('frontend_ssl_mode', 'off') ON CONFLICT (key) DO NOTHING;
INSERT INTO settings (key, value) VALUES ('admin_ssl_mode', 'off') ON CONFLICT (key) DO NOTHING;
-- ACME (Let's Encrypt) email for registration
INSERT INTO settings (key, value) VALUES ('acme_email', '') ON CONFLICT (key) DO NOTHING;
