-- Site logo URL and content license
-- logo_url: URL or data URI for the site's custom logo
-- site_license: license identifier (e.g. "CC-BY-NC-SA-4.0") and custom text

INSERT INTO settings (key, value) VALUES ('site_logo_url', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO settings (key, value) VALUES ('site_license', '') ON CONFLICT (key) DO NOTHING;
INSERT INTO settings (key, value) VALUES ('site_license_url', '') ON CONFLICT (key) DO NOTHING;
