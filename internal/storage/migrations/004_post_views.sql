-- Post views tracking for analytics
-- Tracks both total views and unique visitors per post
CREATE TABLE IF NOT EXISTS post_views (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id INTEGER NOT NULL,
    viewer_hash TEXT NOT NULL,
    viewed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
);

-- Index for efficient queries by post
CREATE INDEX IF NOT EXISTS idx_post_views_post_id ON post_views(post_id);

-- Index for efficient unique visitor queries
CREATE INDEX IF NOT EXISTS idx_post_views_post_viewer ON post_views(post_id, viewer_hash);
