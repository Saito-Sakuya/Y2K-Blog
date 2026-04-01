-- Enable pg_trgm for Chinese-friendly trigram search
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Create trigram GIN indexes for fast ILIKE / similarity search on Chinese text
CREATE INDEX IF NOT EXISTS idx_posts_title_trgm ON posts USING GIN (title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_posts_excerpt_trgm ON posts USING GIN (excerpt gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_posts_tags_trgm ON posts USING GIN (tags gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_posts_content_trgm ON posts USING GIN (content_raw gin_trgm_ops);

-- Update search_vector trigger to use 'simple' config (compatible with CJK)
-- The actual Chinese search will be done via trigram ILIKE, not tsvector
-- Keep tsvector for English/pinyin compatibility
CREATE OR REPLACE FUNCTION posts_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('simple', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('simple', COALESCE(NEW.tags, '')), 'B') ||
        setweight(to_tsvector('simple', COALESCE(NEW.excerpt, '')), 'C') ||
        setweight(to_tsvector('simple', COALESCE(NEW.content_raw, '')), 'D');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
