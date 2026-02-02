package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// PostStatus represents the publication status of a post
type PostStatus string

const (
	StatusDraft     PostStatus = "draft"
	StatusPublished PostStatus = "published"
	StatusScheduled PostStatus = "scheduled"
)

// Post represents a blog post in the database
type Post struct {
	ID          int64
	Slug        string
	Title       string
	Filepath    string
	Status      PostStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedAt *time.Time
	Tags        []string
}

// PostRepository handles database operations for posts
type PostRepository struct {
	db *DB
}

// NewPostRepository creates a new PostRepository
func NewPostRepository(db *DB) *PostRepository {
	return &PostRepository{db: db}
}

// Create inserts a new post into the database
func (r *PostRepository) Create(post *Post) error {
	tagsJSON, err := json.Marshal(post.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	result, err := r.db.Exec(`
		INSERT INTO posts (slug, title, filepath, status, published_at, tags)
		VALUES (?, ?, ?, ?, ?, ?)
	`, post.Slug, post.Title, post.Filepath, post.Status, post.PublishedAt, string(tagsJSON))
	if err != nil {
		return fmt.Errorf("failed to insert post: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}
	post.ID = id

	return nil
}

// Update updates an existing post
func (r *PostRepository) Update(post *Post) error {
	tagsJSON, err := json.Marshal(post.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	_, err = r.db.Exec(`
		UPDATE posts
		SET slug = ?, title = ?, filepath = ?, status = ?,
		    updated_at = CURRENT_TIMESTAMP, published_at = ?, tags = ?
		WHERE id = ?
	`, post.Slug, post.Title, post.Filepath, post.Status, post.PublishedAt, string(tagsJSON), post.ID)
	if err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	return nil
}

// Delete removes a post from the database
func (r *PostRepository) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM posts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}
	return nil
}

// GetByID retrieves a post by its ID
func (r *PostRepository) GetByID(id int64) (*Post, error) {
	post := &Post{}
	var tagsJSON string
	var publishedAt sql.NullTime

	err := r.db.QueryRow(`
		SELECT id, slug, title, filepath, status, created_at, updated_at, published_at, tags
		FROM posts WHERE id = ?
	`, id).Scan(
		&post.ID, &post.Slug, &post.Title, &post.Filepath, &post.Status,
		&post.CreatedAt, &post.UpdatedAt, &publishedAt, &tagsJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	if publishedAt.Valid {
		post.PublishedAt = &publishedAt.Time
	}

	if err := json.Unmarshal([]byte(tagsJSON), &post.Tags); err != nil {
		post.Tags = []string{}
	}

	return post, nil
}

// GetBySlug retrieves a post by its slug
func (r *PostRepository) GetBySlug(slug string) (*Post, error) {
	post := &Post{}
	var tagsJSON string
	var publishedAt sql.NullTime

	err := r.db.QueryRow(`
		SELECT id, slug, title, filepath, status, created_at, updated_at, published_at, tags
		FROM posts WHERE slug = ?
	`, slug).Scan(
		&post.ID, &post.Slug, &post.Title, &post.Filepath, &post.Status,
		&post.CreatedAt, &post.UpdatedAt, &publishedAt, &tagsJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	if publishedAt.Valid {
		post.PublishedAt = &publishedAt.Time
	}

	if err := json.Unmarshal([]byte(tagsJSON), &post.Tags); err != nil {
		post.Tags = []string{}
	}

	return post, nil
}

// ListPublished returns all published posts, ordered by published date descending
func (r *PostRepository) ListPublished(limit, offset int) ([]*Post, error) {
	rows, err := r.db.Query(`
		SELECT id, slug, title, filepath, status, created_at, updated_at, published_at, tags
		FROM posts
		WHERE status = 'published' AND (published_at IS NULL OR published_at <= CURRENT_TIMESTAMP)
		ORDER BY COALESCE(published_at, created_at) DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list posts: %w", err)
	}
	defer rows.Close()

	return r.scanPosts(rows)
}

// ListAll returns all posts regardless of status
func (r *PostRepository) ListAll(limit, offset int) ([]*Post, error) {
	rows, err := r.db.Query(`
		SELECT id, slug, title, filepath, status, created_at, updated_at, published_at, tags
		FROM posts
		ORDER BY COALESCE(published_at, created_at) DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list posts: %w", err)
	}
	defer rows.Close()

	return r.scanPosts(rows)
}

// Search finds posts matching a query in title or tags
func (r *PostRepository) Search(query string, limit int) ([]*Post, error) {
	searchTerm := "%" + query + "%"
	rows, err := r.db.Query(`
		SELECT id, slug, title, filepath, status, created_at, updated_at, published_at, tags
		FROM posts
		WHERE status = 'published'
		  AND (title LIKE ? OR tags LIKE ?)
		ORDER BY COALESCE(published_at, created_at) DESC
		LIMIT ?
	`, searchTerm, searchTerm, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search posts: %w", err)
	}
	defer rows.Close()

	return r.scanPosts(rows)
}

// CountPublished returns the total number of published posts
func (r *PostRepository) CountPublished() (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM posts
		WHERE status = 'published' AND (published_at IS NULL OR published_at <= CURRENT_TIMESTAMP)
	`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count posts: %w", err)
	}
	return count, nil
}

// UpsertBySlug creates or updates a post based on its slug
func (r *PostRepository) UpsertBySlug(post *Post) error {
	existing, err := r.GetBySlug(post.Slug)
	if err != nil {
		return err
	}

	if existing != nil {
		post.ID = existing.ID
		return r.Update(post)
	}

	return r.Create(post)
}

func (r *PostRepository) scanPosts(rows *sql.Rows) ([]*Post, error) {
	var posts []*Post

	for rows.Next() {
		post := &Post{}
		var tagsJSON string
		var publishedAt sql.NullTime

		err := rows.Scan(
			&post.ID, &post.Slug, &post.Title, &post.Filepath, &post.Status,
			&post.CreatedAt, &post.UpdatedAt, &publishedAt, &tagsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}

		if publishedAt.Valid {
			post.PublishedAt = &publishedAt.Time
		}

		if err := json.Unmarshal([]byte(tagsJSON), &post.Tags); err != nil {
			post.Tags = []string{}
		}

		posts = append(posts, post)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating posts: %w", err)
	}

	return posts, nil
}
