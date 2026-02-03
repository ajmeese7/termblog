-- User preferences table for storing theme choices per SSH fingerprint
CREATE TABLE IF NOT EXISTS user_preferences (
    ssh_fingerprint TEXT PRIMARY KEY,
    theme_name TEXT NOT NULL DEFAULT 'pipboy',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
