package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ajmeese7/termblog/internal/blog"
	"github.com/ajmeese7/termblog/internal/storage"
	"github.com/ajmeese7/termblog/internal/theme"
)

// setupTestAPI creates an HTTPServer with a temporary database and test posts
func setupTestAPI(t *testing.T) (*HTTPServer, func()) {
	t.Helper()

	// Create temp directory for test content
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	contentDir := filepath.Join(tmpDir, "posts")
	os.MkdirAll(contentDir, 0o755)

	// Create test post files
	post1 := `---
title: "First Post"
description: "The first test post"
author: "Test Author"
date: 2026-01-15
tags: [go, testing]
draft: false
---

This is the first test post content.
`
	post2 := `---
title: "Second Post"
description: "The second test post"
author: "Test Author"
date: 2026-01-20
tags: [rust, wasm]
draft: false
---

This is the second test post content with more words to increase reading time.
`
	os.WriteFile(filepath.Join(contentDir, "first-post.md"), []byte(post1), 0o644)
	os.WriteFile(filepath.Join(contentDir, "second-post.md"), []byte(post2), 0o644)

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	repo := storage.NewPostRepository(db)
	viewRepo := storage.NewViewRepository(db)
	loader := blog.NewContentLoader(contentDir)

	// Insert test posts
	now := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	now2 := time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)

	p1 := &storage.Post{
		Slug:        "first-post",
		Title:       "First Post",
		Filepath:    filepath.Join(contentDir, "first-post.md"),
		Status:      storage.StatusPublished,
		PublishedAt: &now,
		Tags:        []string{"go", "testing"},
	}
	p2 := &storage.Post{
		Slug:        "second-post",
		Title:       "Second Post",
		Filepath:    filepath.Join(contentDir, "second-post.md"),
		Status:      storage.StatusPublished,
		PublishedAt: &now2,
		Tags:        []string{"rust", "wasm"},
	}

	repo.Create(p1)
	repo.Create(p2)

	// Index posts for FTS
	repo.IndexPost(p1.ID, "First Post", "go testing", "This is the first test post content.")
	repo.IndexPost(p2.ID, "Second Post", "rust wasm", "This is the second test post content with more words.")

	th := theme.DraculaTheme()

	s := &HTTPServer{
		repo:            repo,
		viewRepo:        viewRepo,
		loader:          loader,
		blogTitle:       "Test Blog",
		blogDescription: "A test blog",
		blogAuthor:      "Test Author",
		asciiHeader:     "",
		theme:           th,
		themeKey:        "dracula",
	}

	// Pre-compute cached config response (matches NewHTTPServer behavior)
	if err := s.cacheConfigResponse(); err != nil {
		t.Fatalf("failed to cache config: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return s, cleanup
}

func TestAPIPostsList(t *testing.T) {
	s, cleanup := setupTestAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/posts?page=1&per_page=10", nil)
	w := httptest.NewRecorder()
	s.handleAPIPosts(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result APIPostList
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Total != 2 {
		t.Errorf("expected 2 total posts, got %d", result.Total)
	}
	if len(result.Posts) != 2 {
		t.Errorf("expected 2 posts in response, got %d", len(result.Posts))
	}
	if result.Page != 1 {
		t.Errorf("expected page 1, got %d", result.Page)
	}
}

func TestAPIPostsListPagination(t *testing.T) {
	s, cleanup := setupTestAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/posts?page=1&per_page=1", nil)
	w := httptest.NewRecorder()
	s.handleAPIPosts(w, req)

	var result APIPostList
	json.Unmarshal(w.Body.Bytes(), &result)

	if len(result.Posts) != 1 {
		t.Errorf("expected 1 post per page, got %d", len(result.Posts))
	}
	if result.TotalPages != 2 {
		t.Errorf("expected 2 total pages, got %d", result.TotalPages)
	}
}

func TestAPIPostBySlug(t *testing.T) {
	s, cleanup := setupTestAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/posts/first-post", nil)
	w := httptest.NewRecorder()
	s.handleAPIPostBySlug(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result APIPostDetail
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Slug != "first-post" {
		t.Errorf("expected slug 'first-post', got %q", result.Slug)
	}
	if result.Title != "First Post" {
		t.Errorf("expected title 'First Post', got %q", result.Title)
	}
	if result.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestAPIPostBySlugNotFound(t *testing.T) {
	s, cleanup := setupTestAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/posts/nonexistent", nil)
	w := httptest.NewRecorder()
	s.handleAPIPostBySlug(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAPISearch(t *testing.T) {
	s, cleanup := setupTestAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=first", nil)
	w := httptest.NewRecorder()
	s.handleAPISearch(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result APISearchResult
	json.Unmarshal(w.Body.Bytes(), &result)

	if result.Query != "first" {
		t.Errorf("expected query 'first', got %q", result.Query)
	}
	if len(result.Results) == 0 {
		t.Error("expected at least one search result")
	}
}

func TestAPISearchEmpty(t *testing.T) {
	s, cleanup := setupTestAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=", nil)
	w := httptest.NewRecorder()
	s.handleAPISearch(w, req)

	var result APISearchResult
	json.Unmarshal(w.Body.Bytes(), &result)

	if len(result.Results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(result.Results))
	}
}

func TestAPITags(t *testing.T) {
	s, cleanup := setupTestAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/tags", nil)
	w := httptest.NewRecorder()
	s.handleAPITags(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result []APITag
	json.Unmarshal(w.Body.Bytes(), &result)

	if len(result) == 0 {
		t.Error("expected at least one tag")
	}

	// Should have go, testing, rust, wasm
	tagMap := make(map[string]int)
	for _, tag := range result {
		tagMap[tag.Name] = tag.Count
	}

	if tagMap["go"] != 1 {
		t.Errorf("expected 'go' tag count 1, got %d", tagMap["go"])
	}
	if tagMap["rust"] != 1 {
		t.Errorf("expected 'rust' tag count 1, got %d", tagMap["rust"])
	}
}

func TestAPITagPosts(t *testing.T) {
	s, cleanup := setupTestAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/tags/go", nil)
	w := httptest.NewRecorder()
	s.handleAPITagPosts(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result []APIPost
	json.Unmarshal(w.Body.Bytes(), &result)

	if len(result) != 1 {
		t.Errorf("expected 1 post with tag 'go', got %d", len(result))
	}
}

func TestAPITagPostsNotFound(t *testing.T) {
	s, cleanup := setupTestAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/tags/nonexistent", nil)
	w := httptest.NewRecorder()
	s.handleAPITagPosts(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAPIConfig(t *testing.T) {
	s, cleanup := setupTestAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()
	s.handleAPIConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result APIConfig
	json.Unmarshal(w.Body.Bytes(), &result)

	if result.Title != "Test Blog" {
		t.Errorf("expected title 'Test Blog', got %q", result.Title)
	}
	if result.DefaultTheme != "dracula" {
		t.Errorf("expected default theme 'dracula', got %q", result.DefaultTheme)
	}
	if len(result.Themes) == 0 {
		t.Error("expected at least one theme")
	}
}

func TestAPIConfigETag(t *testing.T) {
	s, cleanup := setupTestAPI(t)
	defer cleanup()

	// First request — should get 200 with ETag
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()
	s.handleAPIConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header")
	}

	// Second request with If-None-Match — should get 304
	req2 := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	req2.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()
	s.handleAPIConfig(w2, req2)

	if w2.Code != http.StatusNotModified {
		t.Fatalf("expected 304, got %d", w2.Code)
	}
	if w2.Body.Len() != 0 {
		t.Error("expected empty body for 304 response")
	}
}

func TestAPIViews(t *testing.T) {
	s, cleanup := setupTestAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/views/first-post", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	s.handleAPIViews(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPIViewsNotFound(t *testing.T) {
	s, cleanup := setupTestAPI(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/views/nonexistent", nil)
	w := httptest.NewRecorder()
	s.handleAPIViews(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAPICORSHeaders(t *testing.T) {
	s, cleanup := setupTestAPI(t)
	defer cleanup()

	// With an allowed origin, CORS headers should be set
	handler := corsMiddleware("https://example.com", s.handleAPIPosts)
	req := httptest.NewRequest(http.MethodOptions, "/api/posts", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Error("missing CORS Allow-Origin header")
	}

	// With empty origin, no CORS headers should be set (same-origin only)
	handler = corsMiddleware("", s.handleAPIPosts)
	req = httptest.NewRequest(http.MethodOptions, "/api/posts", nil)
	w = httptest.NewRecorder()
	handler(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS headers should not be set for empty origin")
	}
}

func TestAPIMethodNotAllowed(t *testing.T) {
	s, cleanup := setupTestAPI(t)
	defer cleanup()

	// POST to a GET-only endpoint
	req := httptest.NewRequest(http.MethodPost, "/api/posts", nil)
	w := httptest.NewRecorder()
	s.handleAPIPosts(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}
