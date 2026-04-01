-- Add status field to posts: 'published', 'draft', 'trashed'
ALTER TABLE posts ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'published';

-- Add trashed_at timestamp for auto-purge logic
ALTER TABLE posts ADD COLUMN IF NOT EXISTS trashed_at TIMESTAMP;

-- Index for filtering by status
CREATE INDEX IF NOT EXISTS idx_posts_status ON posts(status);

-- Set all existing posts to published
UPDATE posts SET status = 'published' WHERE status = '' OR status IS NULL;
