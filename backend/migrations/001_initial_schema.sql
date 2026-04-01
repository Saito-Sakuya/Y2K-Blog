-- Y2K Pixel Blog Database Schema
-- PostgreSQL with zhparser extension for Chinese full-text search

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Boards table
CREATE TABLE IF NOT EXISTS boards (
    id          SERIAL PRIMARY KEY,
    slug        VARCHAR(128) UNIQUE NOT NULL,
    name        VARCHAR(256) NOT NULL,
    color       VARCHAR(16) NOT NULL DEFAULT '#8b7aab',
    icon        VARCHAR(64) NOT NULL DEFAULT 'folder',
    sort_order  INTEGER NOT NULL DEFAULT 0,
    parent_id   INTEGER REFERENCES boards(id) ON DELETE SET NULL,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_boards_parent ON boards(parent_id);
CREATE INDEX idx_boards_order ON boards(sort_order);

-- Posts table (all content types: article, photo, rating, page)
CREATE TABLE IF NOT EXISTS posts (
    id             SERIAL PRIMARY KEY,
    slug           VARCHAR(256) UNIQUE NOT NULL,
    title          VARCHAR(512) NOT NULL,
    type           VARCHAR(16) NOT NULL CHECK (type IN ('article', 'photo', 'rating', 'page')),
    date           DATE NOT NULL DEFAULT CURRENT_DATE,
    tags           TEXT NOT NULL DEFAULT '',         -- comma-separated
    excerpt        TEXT NOT NULL DEFAULT '',
    content_raw    TEXT NOT NULL DEFAULT '',          -- original markdown
    content_html   TEXT NOT NULL DEFAULT '',          -- rendered HTML (sanitized)
    custom_footer  TEXT NOT NULL DEFAULT '',
    read_time      INTEGER NOT NULL DEFAULT 0,       -- minutes
    word_count     INTEGER NOT NULL DEFAULT 0,

    -- Page-specific
    icon           VARCHAR(64) NOT NULL DEFAULT '',
    sort_order     INTEGER NOT NULL DEFAULT 0,
    show_in_menu   BOOLEAN NOT NULL DEFAULT false,

    -- Rating-specific
    cover          VARCHAR(512) NOT NULL DEFAULT '',
    summary        TEXT NOT NULL DEFAULT '',

    -- Photo-specific (pages stored as JSON)
    photo_pages    JSONB,

    -- Rating-specific (radar charts stored as JSON)
    radar_charts   JSONB,

    -- Full-text search vector
    search_vector  TSVECTOR,

    created_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_posts_type ON posts(type);
CREATE INDEX idx_posts_date ON posts(date DESC);
CREATE INDEX idx_posts_slug ON posts(slug);
CREATE INDEX idx_posts_menu ON posts(type, show_in_menu) WHERE type = 'page' AND show_in_menu = true;
CREATE INDEX idx_posts_search ON posts USING GIN(search_vector);

-- Post-Board junction table (many-to-many)
CREATE TABLE IF NOT EXISTS post_boards (
    post_id    INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    board_id   INTEGER NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, board_id)
);

CREATE INDEX idx_post_boards_board ON post_boards(board_id);

-- AI Cache table
CREATE TABLE IF NOT EXISTS ai_cache (
    id            SERIAL PRIMARY KEY,
    slug          VARCHAR(256) NOT NULL,
    title         VARCHAR(512) NOT NULL,
    tags          TEXT NOT NULL DEFAULT '',
    summary_text  TEXT NOT NULL,
    model_used    VARCHAR(128) NOT NULL DEFAULT '',
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_ai_cache_title ON ai_cache(title);

-- Admin users table
CREATE TABLE IF NOT EXISTS users (
    id             SERIAL PRIMARY KEY,
    username       VARCHAR(64) UNIQUE NOT NULL,
    password_hash  VARCHAR(256) NOT NULL,
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Trigger to auto-update search_vector on insert/update
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

CREATE TRIGGER posts_search_vector_trigger
    BEFORE INSERT OR UPDATE ON posts
    FOR EACH ROW
    EXECUTE FUNCTION posts_search_vector_update();

-- Trigger to auto-update updated_at
CREATE OR REPLACE FUNCTION update_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER boards_updated_at BEFORE UPDATE ON boards
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER posts_updated_at BEFORE UPDATE ON posts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- No default admin user: the web setup wizard creates the first admin account.
