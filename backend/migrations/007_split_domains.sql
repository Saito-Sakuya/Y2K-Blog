-- Split domain into frontend_domain + admin_domain
INSERT INTO settings (key, value) VALUES ('frontend_domain', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO settings (key, value) VALUES ('admin_domain', '') ON CONFLICT (key) DO NOTHING;
-- Copy existing domain value to frontend_domain if set
UPDATE settings SET value = (SELECT value FROM settings WHERE key = 'domain')
  WHERE key = 'frontend_domain' AND (SELECT value FROM settings WHERE key = 'domain') != '';
