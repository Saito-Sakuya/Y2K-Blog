-- Add score field for rating posts
ALTER TABLE posts ADD COLUMN IF NOT EXISTS score REAL NOT NULL DEFAULT 0;

-- Index for sort-by-score queries
CREATE INDEX IF NOT EXISTS idx_posts_score ON posts(score DESC) WHERE type = 'rating';
