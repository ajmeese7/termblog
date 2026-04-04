package blog

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Represents the YAML frontmatter of a post
type Frontmatter struct {
	Title        string   `yaml:"title"`
	Description  string   `yaml:"description"`
	Author       string   `yaml:"author"`
	Tags         []string `yaml:"tags"`
	Draft        bool     `yaml:"draft"`
	Date         string   `yaml:"date"`
	PublishedAt  string   `yaml:"published_at"`
	CanonicalURL string   `yaml:"canonical_url"`
}

// Loads and parses markdown content
type ContentLoader struct {
	contentDir string
}

// Creates a new ContentLoader
func NewContentLoader(contentDir string) *ContentLoader {
	return &ContentLoader{contentDir: contentDir}
}

// Loads a single post from a file path.
// The path must be within the configured content directory.
func (l *ContentLoader) LoadPost(filePath string) (*Post, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}
	absDir, err := filepath.Abs(l.contentDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve content dir: %w", err)
	}
	if !strings.HasPrefix(absPath, absDir+string(filepath.Separator)) {
		return nil, fmt.Errorf("path outside content directory")
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return l.ParsePost(string(content), filePath)
}

// Parses a markdown file with frontmatter
func (l *ContentLoader) ParsePost(content string, filePath string) (*Post, error) {
	frontmatter, body, err := l.extractFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Generate slug from filename
	slug := l.slugFromFilename(filePath)

	// Calculate reading time (avg 200 words per minute, minimum 1 minute)
	wordCount := len(strings.Fields(body))
	readingTime := wordCount / 200
	if readingTime < 1 {
		readingTime = 1
	}

	post := &Post{
		Slug:         slug,
		Title:        frontmatter.Title,
		Description:  frontmatter.Description,
		Author:       frontmatter.Author,
		Content:      body,
		Tags:         frontmatter.Tags,
		Draft:        frontmatter.Draft,
		Filepath:     filePath,
		ReadingTime:  readingTime,
		CanonicalURL: frontmatter.CanonicalURL,
	}

	// Parse dates
	if frontmatter.Date != "" {
		if t, err := time.Parse("2006-01-02", frontmatter.Date); err == nil {
			post.CreatedAt = t
		} else if t, err := time.Parse(time.RFC3339, frontmatter.Date); err == nil {
			post.CreatedAt = t
		}
	}

	if frontmatter.PublishedAt != "" {
		if t, err := time.Parse("2006-01-02", frontmatter.PublishedAt); err == nil {
			post.PublishedAt = &t
		} else if t, err := time.Parse(time.RFC3339, frontmatter.PublishedAt); err == nil {
			post.PublishedAt = &t
		}
	} else if !frontmatter.Draft && !post.CreatedAt.IsZero() {
		// If published and no explicit publish date, use created date
		post.PublishedAt = &post.CreatedAt
	}

	// Use filename as title if not specified
	if post.Title == "" {
		post.Title = l.titleFromSlug(slug)
	}

	return post, nil
}

// Loads a post by its slug
func (l *ContentLoader) LoadBySlug(slug string) (*Post, error) {
	posts, err := l.LoadAllPosts()
	if err != nil {
		return nil, err
	}

	for _, post := range posts {
		if post.Slug == slug {
			return post, nil
		}
	}

	return nil, nil
}

// Loads all posts from the content directory
func (l *ContentLoader) LoadAllPosts() ([]*Post, error) {
	var posts []*Post

	err := filepath.Walk(l.contentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		post, err := l.LoadPost(path)
		if err != nil {
			// Log but don't fail on individual post errors
			fmt.Printf("Warning: failed to load %s: %v\n", path, err)
			return nil
		}

		posts = append(posts, post)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk content directory: %w", err)
	}

	return posts, nil
}

// Separates YAML frontmatter from markdown content
func (l *ContentLoader) extractFrontmatter(content string) (*Frontmatter, string, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))

	// Check for opening delimiter
	if !scanner.Scan() {
		return &Frontmatter{}, content, nil
	}

	firstLine := scanner.Text()
	if firstLine != "---" {
		// No frontmatter
		return &Frontmatter{}, content, nil
	}

	// Collect frontmatter lines
	var frontmatterLines []string
	foundClosing := false

	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			foundClosing = true
			break
		}
		frontmatterLines = append(frontmatterLines, line)
	}

	if !foundClosing {
		// Malformed frontmatter, treat whole thing as content
		return &Frontmatter{}, content, nil
	}

	// Parse YAML
	var fm Frontmatter
	frontmatterYAML := strings.Join(frontmatterLines, "\n")
	if err := yaml.Unmarshal([]byte(frontmatterYAML), &fm); err != nil {
		return nil, "", fmt.Errorf("invalid frontmatter YAML: %w", err)
	}

	// Collect remaining content
	var bodyLines []string
	for scanner.Scan() {
		bodyLines = append(bodyLines, scanner.Text())
	}

	body := strings.Join(bodyLines, "\n")
	body = strings.TrimPrefix(body, "\n") // Remove leading newline

	return &fm, body, nil
}

// Generates a URL slug from the filename
func (l *ContentLoader) slugFromFilename(filePath string) string {
	base := filepath.Base(filePath)
	slug := strings.TrimSuffix(base, filepath.Ext(base))

	// Remove date prefix if present (e.g., 2026-02-01-my-post -> my-post)
	datePattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-`)
	slug = datePattern.ReplaceAllString(slug, "")

	// Convert to lowercase and replace spaces/underscores with hyphens
	slug = strings.ToLower(slug)
	slug = strings.ReplaceAll(slug, "_", "-")
	slug = strings.ReplaceAll(slug, " ", "-")

	return slug
}

// Converts a slug to a title
func (l *ContentLoader) titleFromSlug(slug string) string {
	words := strings.Split(slug, "-")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

// Creates a new post file with frontmatter
func (l *ContentLoader) CreatePost(title string, author string) (string, error) {
	slug := l.titleToSlug(title)
	date := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%s-%s.md", date, slug)
	filePath := filepath.Join(l.contentDir, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); err == nil {
		return "", fmt.Errorf("post already exists: %s", filePath)
	}

	content := fmt.Sprintf(`---
title: "%s"
description: ""
author: "%s"
date: %s
tags: []
draft: true
---

Write your post content here...
`, title, author, date)

	// Ensure directory exists
	if err := os.MkdirAll(l.contentDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create content directory: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filePath, nil
}

// Converts a title to a URL-safe slug
func (l *ContentLoader) titleToSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove non-alphanumeric characters except hyphens
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	slug = reg.ReplaceAllString(slug, "")

	// Remove multiple consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Trim leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	return slug
}
