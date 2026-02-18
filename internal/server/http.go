package server

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ajmeese7/termblog/internal/blog"
	"github.com/ajmeese7/termblog/internal/storage"
	"github.com/ajmeese7/termblog/internal/theme"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static
var staticFS embed.FS

//go:embed wasm_dist
var wasmFS embed.FS

// HTTPServer handles HTTP requests and serves the WASM web app
type HTTPServer struct {
	server *http.Server
	repo   *storage.PostRepository
	viewRepo *storage.ViewRepository
	loader *blog.ContentLoader
	feed   *blog.FeedGenerator

	host            string
	port            int
	blogTitle       string
	blogDescription string
	blogAuthor      string
	asciiHeader     string
	theme           *theme.Theme
	themeKey        string // lowercase key matching JS theme map (e.g. "dracula")

	// Cached templates (parsed once at startup)
	templates map[string]*template.Template

	// Cached /api/config response (static for server lifetime)
	configJSON []byte
	configETag string
}

// ThemeColors holds CSS-friendly theme colors for templates
type ThemeColors struct {
	Background string
	Foreground string
	Muted      string
	Accent     string
	Border     string
}

// themeColors returns the theme colors for use in templates
func (s *HTTPServer) themeColors() ThemeColors {
	return ThemeColors{
		Background: s.theme.Colors.Background,
		Foreground: s.theme.Colors.Text,
		Muted:      s.theme.Colors.Muted,
		Accent:     s.theme.Colors.Accent,
		Border:     s.theme.Colors.Border,
	}
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(host string, port int, repo *storage.PostRepository, viewRepo *storage.ViewRepository, loader *blog.ContentLoader, feed *blog.FeedGenerator, blogTitle string, blogDescription string, blogAuthor string, asciiHeader string, t *theme.Theme, themeKey string) (*HTTPServer, error) {
	// Parse and cache templates at startup for efficiency and early error detection
	templates := make(map[string]*template.Template)
	templateNames := []string{"archive.html", "post.html", "tag.html"}

	// Template functions for use in HTML templates
	funcMap := template.FuncMap{
		"lower": strings.ToLower,
	}

	for _, name := range templateNames {
		tmpl, err := template.New(name).Funcs(funcMap).ParseFS(templatesFS, "templates/"+name)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %s: %w", name, err)
		}
		templates[name] = tmpl
	}

	s := &HTTPServer{
		repo:            repo,
		viewRepo:        viewRepo,
		loader:          loader,
		feed:            feed,
		host:            host,
		port:            port,
		blogTitle:       blogTitle,
		blogDescription: blogDescription,
		blogAuthor:      blogAuthor,
		asciiHeader:     asciiHeader,
		theme:           t,
		themeKey:        themeKey,
		templates:       templates,
	}

	// Pre-compute /api/config response and ETag (config is static for server lifetime)
	if err := s.cacheConfigResponse(); err != nil {
		return nil, fmt.Errorf("failed to cache config response: %w", err)
	}

	mux := http.NewServeMux()

	// Static files with long cache (immutable embedded assets)
	staticSub, _ := fs.Sub(staticFS, "static")
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.FS(staticSub)))
	mux.Handle("/static/", cacheControlMiddleware(staticHandler, "public, max-age=31536000, immutable"))

	// JSON API routes
	s.registerAPIRoutes(mux)

	// WASM app at root
	wasmSub, _ := fs.Sub(wasmFS, "wasm_dist")
	wasmHandler := http.FileServer(http.FS(wasmSub))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Serve WASM assets for root and WASM-related files
		if r.URL.Path == "/" || strings.HasSuffix(r.URL.Path, ".wasm") || strings.HasSuffix(r.URL.Path, ".js") {
			// Set correct MIME types for WASM
			if strings.HasSuffix(r.URL.Path, ".wasm") {
				w.Header().Set("Content-Type", "application/wasm")
			}
			wasmHandler.ServeHTTP(w, r)
			return
		}
		// Fall through to 404 for other paths
		http.NotFound(w, r)
	})
	mux.HandleFunc("/feed.xml", s.handleRSSFeed)
	mux.HandleFunc("/feed.json", s.handleJSONFeed)
	mux.HandleFunc("/sitemap.xml", s.handleSitemap)
	mux.HandleFunc("/robots.txt", s.handleRobots)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/archive", s.handleArchive)
	mux.HandleFunc("/posts/", s.handlePost)
	mux.HandleFunc("/tags/", s.handleTag)

	// Wrap mux with gzip compression middleware
	handler := GzipMiddleware(mux)

	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s, nil
}

// handleRSSFeed serves the RSS feed
func (s *HTTPServer) handleRSSFeed(w http.ResponseWriter, r *http.Request) {
	posts, err := s.repo.ListPublished(50, 0)
	if err != nil {
		log.Printf("Failed to load posts for feed: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Convert storage posts to blog posts
	var blogPosts []*blog.Post
	for _, p := range posts {
		blogPosts = append(blogPosts, &blog.Post{
			Slug:        p.Slug,
			Title:       p.Title,
			Tags:        p.Tags,
			PublishedAt: p.PublishedAt,
			CreatedAt:   p.CreatedAt,
		})
	}

	rss, err := s.feed.GenerateRSS(blogPosts)
	if err != nil {
		log.Printf("Failed to generate RSS: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	w.Write([]byte(rss))
}

// handleJSONFeed serves the JSON feed
func (s *HTTPServer) handleJSONFeed(w http.ResponseWriter, r *http.Request) {
	posts, err := s.repo.ListPublished(50, 0)
	if err != nil {
		log.Printf("Failed to load posts for feed: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var blogPosts []*blog.Post
	for _, p := range posts {
		blogPosts = append(blogPosts, &blog.Post{
			Slug:        p.Slug,
			Title:       p.Title,
			Tags:        p.Tags,
			PublishedAt: p.PublishedAt,
			CreatedAt:   p.CreatedAt,
		})
	}

	jsonFeed, err := s.feed.GenerateJSON(blogPosts)
	if err != nil {
		log.Printf("Failed to generate JSON feed: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/feed+json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	w.Write([]byte(jsonFeed))
}

// handleHealth serves a health check endpoint
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// handleSitemap generates and serves an XML sitemap
func (s *HTTPServer) handleSitemap(w http.ResponseWriter, r *http.Request) {
	posts, err := s.repo.ListPublished(1000, 0)
	if err != nil {
		log.Printf("Failed to load posts for sitemap: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Convert storage posts to blog posts
	var blogPosts []*blog.Post
	for _, p := range posts {
		blogPosts = append(blogPosts, &blog.Post{
			Slug:        p.Slug,
			Title:       p.Title,
			Tags:        p.Tags,
			PublishedAt: p.PublishedAt,
			CreatedAt:   p.CreatedAt,
		})
	}

	// Use feed's base URL for sitemap
	sitemapGen := blog.NewSitemapGenerator(s.feed.BaseURL())
	sitemap, err := sitemapGen.Generate(blogPosts)
	if err != nil {
		log.Printf("Failed to generate sitemap: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	w.Write([]byte(sitemap))
}

// handleRobots serves the robots.txt file
func (s *HTTPServer) handleRobots(w http.ResponseWriter, r *http.Request) {
	baseURL := s.feed.BaseURL()
	robots := fmt.Sprintf(`User-agent: *
Allow: /

Sitemap: %s/sitemap.xml
`, baseURL)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 24 hours
	w.Write([]byte(robots))
}

// handleArchive serves the archive page with all posts
func (s *HTTPServer) handleArchive(w http.ResponseWriter, r *http.Request) {
	posts, err := s.repo.ListPublished(1000, 0)
	if err != nil {
		log.Printf("Failed to load posts for archive: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Load full post data including reading time
	var blogPosts []*blog.Post
	for _, p := range posts {
		post, err := s.loader.LoadPost(p.Filepath)
		if err != nil {
			log.Printf("Failed to load post %s: %v", p.Filepath, err)
			continue
		}
		blogPosts = append(blogPosts, post)
	}

	data := struct {
		BlogTitle       string
		BlogDescription string
		BaseURL         string
		Posts           []*blog.Post
		Theme           ThemeColors
	}{
		BlogTitle:       s.blogTitle,
		BlogDescription: s.blogDescription,
		BaseURL:         s.feed.BaseURL(),
		Posts:           blogPosts,
		Theme:           s.themeColors(),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes
	if err := s.templates["archive.html"].Execute(w, data); err != nil {
		log.Printf("Failed to execute archive template: %v", err)
	}
}

// handlePost serves individual post pages
func (s *HTTPServer) handlePost(w http.ResponseWriter, r *http.Request) {
	// Handle /posts and /posts/ -> redirect to archive
	if r.URL.Path == "/posts" || r.URL.Path == "/posts/" {
		http.Redirect(w, r, "/archive", http.StatusFound)
		return
	}

	// Extract slug from URL path /posts/{slug}
	slug := strings.TrimPrefix(r.URL.Path, "/posts/")
	slug = strings.TrimSuffix(slug, "/")

	if slug == "" {
		http.Redirect(w, r, "/archive", http.StatusFound)
		return
	}

	// Look up post by slug
	dbPost, err := s.repo.GetBySlug(slug)
	if err != nil {
		log.Printf("Failed to get post %s: %v", slug, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if dbPost == nil {
		http.NotFound(w, r)
		return
	}

	// Load full post content
	post, err := s.loader.LoadPost(dbPost.Filepath)
	if err != nil {
		log.Printf("Failed to load post content %s: %v", slug, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Render markdown to HTML
	htmlContent, err := blog.RenderMarkdownToHTML(post.Content)
	if err != nil {
		log.Printf("Failed to render post %s: %v", slug, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := struct {
		BlogTitle string
		BaseURL   string
		Post      *blog.Post
		Content   template.HTML
		Theme     ThemeColors
	}{
		BlogTitle: s.blogTitle,
		BaseURL:   s.feed.BaseURL(),
		Post:      post,
		Content:   template.HTML(htmlContent),
		Theme:     s.themeColors(),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	if err := s.templates["post.html"].Execute(w, data); err != nil {
		log.Printf("Failed to execute post template: %v", err)
	}
}

// handleTag serves tag listing pages
func (s *HTTPServer) handleTag(w http.ResponseWriter, r *http.Request) {
	// Extract tag from URL path /tags/{tag} and normalize to lowercase
	tag := strings.TrimPrefix(r.URL.Path, "/tags/")
	tag = strings.TrimSuffix(tag, "/")
	tag = strings.ToLower(tag)

	if tag == "" {
		http.Redirect(w, r, "/archive", http.StatusFound)
		return
	}

	// Get all published posts and filter by tag
	allPosts, err := s.repo.ListPublished(1000, 0)
	if err != nil {
		log.Printf("Failed to load posts for tag %s: %v", tag, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var blogPosts []*blog.Post
	for _, p := range allPosts {
		// Check if post has this tag (case-insensitive)
		for _, t := range p.Tags {
			if strings.EqualFold(t, tag) {
				post, err := s.loader.LoadPost(p.Filepath)
				if err != nil {
					log.Printf("Failed to load post %s: %v", p.Filepath, err)
					break
				}
				blogPosts = append(blogPosts, post)
				break
			}
		}
	}

	// Return 404 if no posts found with this tag
	if len(blogPosts) == 0 {
		http.NotFound(w, r)
		return
	}

	data := struct {
		BlogTitle string
		BaseURL   string
		Tag       string
		Posts     []*blog.Post
		Theme     ThemeColors
	}{
		BlogTitle: s.blogTitle,
		BaseURL:   s.feed.BaseURL(),
		Tag:       tag,
		Posts:     blogPosts,
		Theme:     s.themeColors(),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes
	if err := s.templates["tag.html"].Execute(w, data); err != nil {
		log.Printf("Failed to execute tag template: %v", err)
	}
}

// cacheControlMiddleware wraps an http.Handler to add Cache-Control headers
func cacheControlMiddleware(next http.Handler, cacheControl string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", cacheControl)
		next.ServeHTTP(w, r)
	})
}

// cacheConfigResponse pre-computes the /api/config JSON response and ETag.
// Config is static for the server's lifetime, so we compute it once at startup.
func (s *HTTPServer) cacheConfigResponse() error {
	themes := theme.DefaultThemes()
	themeInfos := make([]APIThemeInfo, 0, len(themes))
	for key, t := range themes {
		themeInfos = append(themeInfos, APIThemeInfo{
			Key:         key,
			Name:        t.Name,
			Description: t.Description,
			Colors: APIThemeColors{
				Primary:    t.Colors.Primary,
				Secondary:  t.Colors.Secondary,
				Background: t.Colors.Background,
				Text:       t.Colors.Text,
				Muted:      t.Colors.Muted,
				Accent:     t.Colors.Accent,
				Error:      t.Colors.Error,
				Success:    t.Colors.Success,
				Warning:    t.Colors.Warning,
				Border:     t.Colors.Border,
			},
		})
	}

	cfg := APIConfig{
		Title:        s.blogTitle,
		Description:  s.blogDescription,
		Author:       s.blogAuthor,
		Themes:       themeInfos,
		DefaultTheme: s.themeKey,
		ASCIIHeader:  s.asciiHeader,
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	s.configJSON = data
	hash := sha256.Sum256(data)
	s.configETag = fmt.Sprintf(`"%x"`, hash[:8])
	return nil
}

// Start starts the HTTP server
func (s *HTTPServer) Start() error {
	log.Printf("HTTP server starting on %s:%d", s.host, s.port)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// ListenAndServeWithSignal starts the server and handles shutdown signals
func (s *HTTPServer) ListenAndServeWithSignal() error {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	log.Printf("HTTP server starting on %s:%d", s.host, s.port)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	<-done
	log.Println("HTTP server shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}
