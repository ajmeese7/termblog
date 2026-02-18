package server

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// API response types

// APIPost is a post summary for list endpoints
type APIPost struct {
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Author      string   `json:"author,omitempty"`
	Tags        []string `json:"tags"`
	PublishedAt string   `json:"published_at,omitempty"`
	ReadingTime int      `json:"reading_time"`
}

// APIPostDetail is a full post with markdown content
type APIPostDetail struct {
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Author      string   `json:"author,omitempty"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags"`
	PublishedAt string   `json:"published_at,omitempty"`
	ReadingTime int      `json:"reading_time"`
}

// APIPostList is a paginated list of posts
type APIPostList struct {
	Posts      []APIPost `json:"posts"`
	Total      int       `json:"total"`
	Page       int       `json:"page"`
	PerPage    int       `json:"per_page"`
	TotalPages int       `json:"total_pages"`
}

// APISearchResult wraps search results
type APISearchResult struct {
	Query   string    `json:"query"`
	Results []APIPost `json:"results"`
	Total   int       `json:"total"`
}

// APITag represents a tag with its post count
type APITag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// APIConfig exposes blog configuration to the WASM app
type APIConfig struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Author      string          `json:"author"`
	Themes      []APIThemeInfo  `json:"themes"`
	DefaultTheme string         `json:"default_theme"`
	ASCIIHeader string          `json:"ascii_header,omitempty"`
}

// APIThemeInfo describes a theme for the client
type APIThemeInfo struct {
	Key         string         `json:"key"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Colors      APIThemeColors `json:"colors"`
}

// APIThemeColors holds all theme color values
type APIThemeColors struct {
	Primary    string `json:"primary"`
	Secondary  string `json:"secondary"`
	Background string `json:"background"`
	Text       string `json:"text"`
	Muted      string `json:"muted"`
	Accent     string `json:"accent"`
	Error      string `json:"error"`
	Success    string `json:"success"`
	Warning    string `json:"warning"`
	Border     string `json:"border"`
}

// corsMiddleware adds CORS headers for local development
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

// writeError writes a JSON error response
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// handleAPIPosts handles GET /api/posts?page=1&per_page=10
func (s *HTTPServer) handleAPIPosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 10
	}

	total, err := s.repo.CountPublished()
	if err != nil {
		log.Printf("API: failed to count posts: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	offset := (page - 1) * perPage
	dbPosts, err := s.repo.ListPublished(perPage, offset)
	if err != nil {
		log.Printf("API: failed to list posts: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	posts := make([]APIPost, 0, len(dbPosts))
	for _, p := range dbPosts {
		ap := APIPost{
			Slug:  p.Slug,
			Title: p.Title,
			Tags:  p.Tags,
		}
		if ap.Tags == nil {
			ap.Tags = []string{}
		}
		if p.PublishedAt != nil {
			ap.PublishedAt = p.PublishedAt.Format("2006-01-02")
		}
		// Load full post for description and reading time
		if post, err := s.loader.LoadPost(p.Filepath); err == nil {
			ap.Description = post.Description
			ap.Author = post.Author
			ap.ReadingTime = post.ReadingTime
		}
		posts = append(posts, ap)
	}

	totalPages := total / perPage
	if total%perPage != 0 {
		totalPages++
	}

	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, http.StatusOK, APIPostList{
		Posts:      posts,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	})
}

// handleAPIPostBySlug handles GET /api/posts/{slug}
func (s *HTTPServer) handleAPIPostBySlug(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	slug := strings.TrimPrefix(r.URL.Path, "/api/posts/")
	slug = strings.TrimSuffix(slug, "/")
	if slug == "" {
		// No slug — delegate to list handler
		s.handleAPIPosts(w, r)
		return
	}

	dbPost, err := s.repo.GetBySlug(slug)
	if err != nil {
		log.Printf("API: failed to get post %s: %v", slug, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if dbPost == nil || dbPost.Status != "published" {
		writeError(w, http.StatusNotFound, "post not found")
		return
	}

	post, err := s.loader.LoadPost(dbPost.Filepath)
	if err != nil {
		log.Printf("API: failed to load post %s: %v", slug, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	detail := APIPostDetail{
		Slug:        post.Slug,
		Title:       post.Title,
		Description: post.Description,
		Author:      post.Author,
		Content:     post.Content,
		Tags:        post.Tags,
		ReadingTime: post.ReadingTime,
	}
	if detail.Tags == nil {
		detail.Tags = []string{}
	}
	if dbPost.PublishedAt != nil {
		detail.PublishedAt = dbPost.PublishedAt.Format("2006-01-02")
	}

	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, detail)
}

// handleAPISearch handles GET /api/search?q=query&limit=20
func (s *HTTPServer) handleAPISearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, http.StatusOK, APISearchResult{
			Query:   "",
			Results: []APIPost{},
			Total:   0,
		})
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	dbPosts, err := s.repo.Search(query, limit)
	if err != nil {
		log.Printf("API: search failed for %q: %v", query, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	results := make([]APIPost, 0, len(dbPosts))
	for _, p := range dbPosts {
		ap := APIPost{
			Slug:  p.Slug,
			Title: p.Title,
			Tags:  p.Tags,
		}
		if ap.Tags == nil {
			ap.Tags = []string{}
		}
		if p.PublishedAt != nil {
			ap.PublishedAt = p.PublishedAt.Format("2006-01-02")
		}
		if post, err := s.loader.LoadPost(p.Filepath); err == nil {
			ap.Description = post.Description
			ap.Author = post.Author
			ap.ReadingTime = post.ReadingTime
		}
		results = append(results, ap)
	}

	writeJSON(w, http.StatusOK, APISearchResult{
		Query:   query,
		Results: results,
		Total:   len(results),
	})
}

// handleAPITags handles GET /api/tags
func (s *HTTPServer) handleAPITags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	posts, err := s.repo.ListPublished(1000, 0)
	if err != nil {
		log.Printf("API: failed to list posts for tags: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	tagCounts := make(map[string]int)
	for _, p := range posts {
		for _, t := range p.Tags {
			tagCounts[strings.ToLower(t)]++
		}
	}

	tags := make([]APITag, 0, len(tagCounts))
	for name, count := range tagCounts {
		tags = append(tags, APITag{Name: name, Count: count})
	}

	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, tags)
}

// handleAPITagPosts handles GET /api/tags/{tag}
func (s *HTTPServer) handleAPITagPosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	tag := strings.TrimPrefix(r.URL.Path, "/api/tags/")
	tag = strings.TrimSuffix(tag, "/")
	tag = strings.ToLower(tag)

	if tag == "" {
		s.handleAPITags(w, r)
		return
	}

	allPosts, err := s.repo.ListPublished(1000, 0)
	if err != nil {
		log.Printf("API: failed to list posts for tag %s: %v", tag, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	posts := make([]APIPost, 0)
	for _, p := range allPosts {
		for _, t := range p.Tags {
			if strings.EqualFold(t, tag) {
				ap := APIPost{
					Slug:  p.Slug,
					Title: p.Title,
					Tags:  p.Tags,
				}
				if ap.Tags == nil {
					ap.Tags = []string{}
				}
				if p.PublishedAt != nil {
					ap.PublishedAt = p.PublishedAt.Format("2006-01-02")
				}
				if post, err := s.loader.LoadPost(p.Filepath); err == nil {
					ap.Description = post.Description
					ap.Author = post.Author
					ap.ReadingTime = post.ReadingTime
				}
				posts = append(posts, ap)
				break
			}
		}
	}

	if len(posts) == 0 {
		writeError(w, http.StatusNotFound, "tag not found")
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, posts)
}

// handleAPIConfig handles GET /api/config
func (s *HTTPServer) handleAPIConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Serve cached response with ETag support
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("ETag", s.configETag)

	if match := r.Header.Get("If-None-Match"); match == s.configETag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(s.configJSON)
}

// handleAPIViews handles POST /api/views/{slug}
func (s *HTTPServer) handleAPIViews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	slug := strings.TrimPrefix(r.URL.Path, "/api/views/")
	slug = strings.TrimSuffix(slug, "/")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "slug required")
		return
	}

	dbPost, err := s.repo.GetBySlug(slug)
	if err != nil {
		log.Printf("API: failed to get post %s for view: %v", slug, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if dbPost == nil {
		writeError(w, http.StatusNotFound, "post not found")
		return
	}

	// Hash the client IP for privacy
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = strings.Split(forwarded, ",")[0]
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(ip)))

	if err := s.viewRepo.RecordView(dbPost.ID, hash[:16]); err != nil {
		log.Printf("API: failed to record view for %s: %v", slug, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// registerAPIRoutes registers all /api/* routes on the given mux
func (s *HTTPServer) registerAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/posts/", corsMiddleware(s.handleAPIPostBySlug))
	mux.HandleFunc("/api/posts", corsMiddleware(s.handleAPIPosts))
	mux.HandleFunc("/api/search", corsMiddleware(s.handleAPISearch))
	mux.HandleFunc("/api/tags/", corsMiddleware(s.handleAPITagPosts))
	mux.HandleFunc("/api/tags", corsMiddleware(s.handleAPITags))
	mux.HandleFunc("/api/config", corsMiddleware(s.handleAPIConfig))
	mux.HandleFunc("/api/views/", corsMiddleware(s.handleAPIViews))
}
