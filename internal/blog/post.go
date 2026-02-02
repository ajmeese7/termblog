package blog

import (
	"time"
)

// Post represents a parsed blog post with content
type Post struct {
	Slug        string
	Title       string
	Description string
	Author      string
	Content     string
	Tags        []string
	Draft       bool
	CreatedAt   time.Time
	PublishedAt *time.Time
	Filepath    string
}
