-- Per-post custom CSS field and toggle
ALTER TABLE posts ADD COLUMN IF NOT EXISTS custom_css TEXT DEFAULT '';
ALTER TABLE posts ADD COLUMN IF NOT EXISTS css_enabled BOOLEAN NOT NULL DEFAULT false;

-- Site-level settings (will be set via settings API)
-- custom_css_enabled_types: comma-separated types, e.g. "article,photo,rating,page"
-- global_css: CSS applied to all pages
INSERT INTO settings (key, value) VALUES ('custom_css_enabled_types', 'article,photo,rating,page') ON CONFLICT (key) DO NOTHING;
INSERT INTO settings (key, value) VALUES ('global_css', '') ON CONFLICT (key) DO NOTHING;
