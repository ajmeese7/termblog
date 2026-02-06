package storage

import (
	"time"
)

// ViewRepository handles database operations for post views
type ViewRepository struct {
	db *DB
}

// NewViewRepository creates a new ViewRepository
func NewViewRepository(db *DB) *ViewRepository {
	return &ViewRepository{db: db}
}

// RecordView records a view for a post
// viewerHash should be SSH fingerprint, session ID, or IP hash
func (r *ViewRepository) RecordView(postID int64, viewerHash string) error {
	_, err := r.db.Exec(`
		INSERT INTO post_views (post_id, viewer_hash)
		VALUES (?, ?)
	`, postID, viewerHash)
	return err
}

// ViewStats holds view statistics for a post
type ViewStats struct {
	PostID        int64
	TotalViews    int
	UniqueViewers int
}

// GetViewStats returns view statistics for a post
func (r *ViewRepository) GetViewStats(postID int64) (*ViewStats, error) {
	var stats ViewStats
	stats.PostID = postID

	// Get total views
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM post_views WHERE post_id = ?
	`, postID).Scan(&stats.TotalViews)
	if err != nil {
		return nil, err
	}

	// Get unique viewers
	err = r.db.QueryRow(`
		SELECT COUNT(DISTINCT viewer_hash) FROM post_views WHERE post_id = ?
	`, postID).Scan(&stats.UniqueViewers)
	if err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetAllViewStats returns view statistics for all posts
func (r *ViewRepository) GetAllViewStats() (map[int64]*ViewStats, error) {
	rows, err := r.db.Query(`
		SELECT
			post_id,
			COUNT(*) as total_views,
			COUNT(DISTINCT viewer_hash) as unique_viewers
		FROM post_views
		GROUP BY post_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[int64]*ViewStats)
	for rows.Next() {
		var s ViewStats
		if err := rows.Scan(&s.PostID, &s.TotalViews, &s.UniqueViewers); err != nil {
			return nil, err
		}
		stats[s.PostID] = &s
	}

	return stats, rows.Err()
}

// PopularPost represents a post with its view count
type PopularPost struct {
	PostID        int64
	TotalViews    int
	UniqueViewers int
}

// GetPopularPosts returns the most viewed posts
func (r *ViewRepository) GetPopularPosts(limit int) ([]*PopularPost, error) {
	rows, err := r.db.Query(`
		SELECT
			post_id,
			COUNT(*) as total_views,
			COUNT(DISTINCT viewer_hash) as unique_viewers
		FROM post_views
		GROUP BY post_id
		ORDER BY total_views DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*PopularPost
	for rows.Next() {
		var p PopularPost
		if err := rows.Scan(&p.PostID, &p.TotalViews, &p.UniqueViewers); err != nil {
			return nil, err
		}
		posts = append(posts, &p)
	}

	return posts, rows.Err()
}

// GetRecentViewers returns recent viewers for a post (for debugging/admin)
func (r *ViewRepository) GetRecentViewers(postID int64, limit int) ([]time.Time, error) {
	rows, err := r.db.Query(`
		SELECT viewed_at FROM post_views
		WHERE post_id = ?
		ORDER BY viewed_at DESC
		LIMIT ?
	`, postID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var times []time.Time
	for rows.Next() {
		var t time.Time
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		times = append(times, t)
	}

	return times, rows.Err()
}
