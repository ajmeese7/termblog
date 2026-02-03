package storage

import (
	"database/sql"
	"fmt"
)

// PreferenceRepository handles database operations for user preferences
type PreferenceRepository struct {
	db *DB
}

// NewPreferenceRepository creates a new PreferenceRepository
func NewPreferenceRepository(db *DB) *PreferenceRepository {
	return &PreferenceRepository{db: db}
}

// GetTheme retrieves the saved theme for a user (identified by SSH fingerprint)
func (r *PreferenceRepository) GetTheme(fingerprint string) (string, error) {
	if fingerprint == "" {
		return "pipboy", nil
	}

	var theme string
	err := r.db.QueryRow(
		"SELECT theme_name FROM user_preferences WHERE ssh_fingerprint = ?",
		fingerprint,
	).Scan(&theme)

	if err == sql.ErrNoRows {
		return "pipboy", nil // default theme
	}
	if err != nil {
		return "pipboy", fmt.Errorf("failed to get theme: %w", err)
	}

	return theme, nil
}

// SetTheme saves the theme preference for a user
func (r *PreferenceRepository) SetTheme(fingerprint, themeName string) error {
	if fingerprint == "" {
		return nil // Can't save without a fingerprint
	}

	_, err := r.db.Exec(`
		INSERT INTO user_preferences (ssh_fingerprint, theme_name, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(ssh_fingerprint) DO UPDATE SET
			theme_name = excluded.theme_name,
			updated_at = CURRENT_TIMESTAMP
	`, fingerprint, themeName)

	if err != nil {
		return fmt.Errorf("failed to set theme: %w", err)
	}

	return nil
}
