-- Add reading time column to posts
ALTER TABLE posts ADD COLUMN reading_time INTEGER DEFAULT 1;
