-- Full-text search index for posts
-- Uses FTS5 for efficient text search across titles, tags, and content
CREATE VIRTUAL TABLE IF NOT EXISTS posts_fts USING fts5(
    title,
    tags,
    content,
    content_id UNINDEXED,
    tokenize='porter unicode61'
);
