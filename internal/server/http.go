package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/base64"
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

	"github.com/ajmeese7/termblog/internal/app"
	"github.com/ajmeese7/termblog/internal/blog"
	"github.com/ajmeese7/termblog/internal/storage"
	"github.com/ajmeese7/termblog/internal/theme"
)

type ctxKey int

const cspNonceKey ctxKey = iota

// Generates a fresh CSP nonce. 16 random bytes is plenty for a per-request nonce.
func generateCSPNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// Returns the per-request CSP nonce, or an empty string if the context lacks one
// (e.g. test paths that bypass the middleware).
func nonceFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(cspNonceKey).(string); ok {
		return v
	}
	return ""
}

//go:embed templates/*
var templatesFS embed.FS

//go:embed static
var staticFS embed.FS

//go:embed wasm_dist
var wasmFS embed.FS

// Handles HTTP requests and serves the WASM web app
type HTTPServer struct {
	server   *http.Server
	repo     *storage.PostRepository
	viewRepo *storage.ViewRepository
	loader   *blog.ContentLoader
	feed     *blog.FeedGenerator

	host            string
	port            int
	blogTitle       string
	blogDescription string
	blogAuthor      string
	asciiHeader     string
	theme           *theme.Theme
	themeKey        string // lowercase key matching JS theme map (e.g. "dracula")

	// Whether to trust X-Forwarded-For headers (only enable behind a reverse proxy)
	trustProxy bool

	// Cached templates (parsed once at startup)
	templates map[string]*template.Template

	// Cached /api/config response (static for server lifetime)
	configJSON []byte
	configETag string

	// Cached WASM index.html bytes; CSP nonces are injected per request.
	indexHTML []byte

	// Favicon configuration and pre-rendered SVGs (one per theme key) for
	// letter and emoji modes. Image mode populates faviconImageOrigin instead
	// so the CSP middleware can allow the cross-origin fetch.
	faviconCfg          app.FaviconConfig
	faviconRendered     map[string]*faviconResource
	faviconImageOrigin  string
	faviconExtraImgSrc  string // pre-formatted addition to img-src CSP directive
}

// faviconResource caches the bytes of a generated SVG favicon along with its
// ETag, so repeat requests skip the (already cheap) re-render.
type faviconResource struct {
	body []byte
	etag string
}

// Holds CSS-friendly theme colors for templates
type ThemeColors struct {
	Background string
	Foreground string
	Muted      string
	Accent     string
	Border     string
}

// Returns the theme colors for use in templates
func (s *HTTPServer) themeColors() ThemeColors {
	return ThemeColors{
		Background: s.theme.Colors.Background,
		Foreground: s.theme.Colors.Text,
		Muted:      s.theme.Colors.Muted,
		Accent:     s.theme.Colors.Accent,
		Border:     s.theme.Colors.Border,
	}
}

// Creates a new HTTP server
func NewHTTPServer(
	host string,
	port int,
	repo *storage.PostRepository,
	viewRepo *storage.ViewRepository,
	loader *blog.ContentLoader,
	feed *blog.FeedGenerator,
	blogTitle string,
	blogDescription string,
	blogAuthor string,
	asciiHeader string,
	t *theme.Theme,
	themeKey string,
	trustProxy bool,
	faviconCfg app.FaviconConfig,
) (*HTTPServer, error) {
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
		trustProxy:      trustProxy,
		templates:       templates,
		faviconCfg:      faviconCfg,
	}

	// Pre-render favicon SVGs for letter/emoji modes (cheap and gives us a
	// stable ETag at boot). Image mode pulls bytes off disk per request via
	// http.ServeFile, or redirects to the configured URL.
	if err := s.prepareFavicon(); err != nil {
		return nil, fmt.Errorf("failed to prepare favicon: %w", err)
	}

	// Pre-compute /api/config response and ETag (config is static for server lifetime)
	if err := s.cacheConfigResponse(); err != nil {
		return nil, fmt.Errorf("failed to cache config response: %w", err)
	}

	// Cache the WASM index.html bytes so we can inject a CSP nonce per request
	// without re-reading the embedded FS each time. The favicon meta+link
	// substitution happens once here so per-request handling stays cheap.
	indexBytes, err := fs.ReadFile(wasmFS, "wasm_dist/index.html")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded index.html: %w", err)
	}
	s.indexHTML = injectFaviconHead(indexBytes, s.faviconCfg)

	mux := http.NewServeMux()

	// Static files with long cache (immutable embedded assets)
	staticSub, _ := fs.Sub(staticFS, "static")
	staticHandler := http.StripPrefix("/static/", http.FileServer(http.FS(staticSub)))
	mux.Handle("/static/", cacheControlMiddleware(staticHandler, "public, max-age=31536000, immutable"))

	// JSON API routes (rate-limited: 60 requests per minute per IP)
	httpLimiter := NewRateLimiter(60, time.Minute)
	s.registerAPIRoutes(mux, httpLimiter)

	// WASM app at root
	wasmSub, _ := fs.Sub(wasmFS, "wasm_dist")
	wasmHandler := http.FileServer(http.FS(wasmSub))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Serve the index with a freshly-injected CSP nonce so trunk's inline
		// loader and the theme-prefetch script can run under a strict CSP.
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			s.handleIndex(w, r)
			return
		}
		// Static WASM assets (hashed filenames) can be served straight from the FS.
		if strings.HasSuffix(r.URL.Path, ".wasm") || strings.HasSuffix(r.URL.Path, ".js") {
			if strings.HasSuffix(r.URL.Path, ".wasm") {
				w.Header().Set("Content-Type", "application/wasm")
			}
			wasmHandler.ServeHTTP(w, r)
			return
		}
		// Fall through to 404 for other paths
		http.NotFound(w, r)
	})
	mux.HandleFunc("/favicon", s.handleFavicon)
	mux.HandleFunc("/favicon.ico", s.handleFavicon)
	mux.HandleFunc("/feed.xml", s.handleRSSFeed)
	mux.HandleFunc("/feed.json", s.handleJSONFeed)
	mux.HandleFunc("/sitemap.xml", s.handleSitemap)
	mux.HandleFunc("/robots.txt", s.handleRobots)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/archive", s.handleArchive)
	mux.HandleFunc("/posts/", s.handlePost)
	mux.HandleFunc("/tags/", s.handleTag)

	// Wrap mux with security headers and gzip compression middleware
	handler := s.securityHeadersMiddleware(GzipMiddleware(mux))

	s.server = &http.Server{
		MaxHeaderBytes: 1 << 20, // 1MB max header size
		Addr:           fmt.Sprintf("%s:%d", host, port),
		Handler:        handler,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
	}

	return s, nil
}

// Serves the RSS feed
func (s *HTTPServer) handleRSSFeed(w http.ResponseWriter, r *http.Request) {
	posts, err := s.repo.ListPublished(50, 0)
	if err != nil {
		log.Printf("Failed to load posts for feed: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Load full posts
	var blogPosts []*blog.Post
	for _, p := range posts {
		if post, err := s.loader.LoadPost(p.Filepath); err == nil {
			blogPosts = append(blogPosts, post)
		}
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

// Serves the JSON feed
func (s *HTTPServer) handleJSONFeed(w http.ResponseWriter, r *http.Request) {
	posts, err := s.repo.ListPublished(50, 0)
	if err != nil {
		log.Printf("Failed to load posts for feed: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var blogPosts []*blog.Post
	for _, p := range posts {
		if post, err := s.loader.LoadPost(p.Filepath); err == nil {
			blogPosts = append(blogPosts, post)
		}
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

// Serves a health check endpoint
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// Generates and serves an XML sitemap
func (s *HTTPServer) handleSitemap(w http.ResponseWriter, r *http.Request) {
	posts, err := s.repo.ListPublished(1000, 0)
	if err != nil {
		log.Printf("Failed to load posts for sitemap: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Load full posts
	var blogPosts []*blog.Post
	for _, p := range posts {
		if post, err := s.loader.LoadPost(p.Filepath); err == nil {
			blogPosts = append(blogPosts, post)
		}
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

// Serves the robots.txt file
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

// Serves the archive page with all posts
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

// Serves individual post pages
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

// Serves tag listing pages
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

// Adds security headers to all HTTP responses. A fresh CSP nonce is generated
// per request and stashed in the request context so HTML handlers can mark
// their inline <script> tags as authorized.
//
// When favicon.mode is "image" with an http(s) URL, the URL's origin is added
// to img-src so the cross-origin favicon redirect actually loads.
func (s *HTTPServer) securityHeadersMiddleware(next http.Handler) http.Handler {
	imgSrc := "'self' data:" + s.faviconExtraImgSrc
	csp := "default-src 'self'; script-src 'self' 'wasm-unsafe-eval' 'nonce-%s'; style-src 'self' 'unsafe-inline'; img-src " + imgSrc + "; connect-src 'self'; frame-ancestors 'none'"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce, err := generateCSPNonce()
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", fmt.Sprintf(csp, nonce))
		ctx := context.WithValue(r.Context(), cspNonceKey, nonce)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Serves the WASM app index.html with a CSP nonce injected into every <script>
// tag. The HTML cannot be cached because each response carries a fresh nonce.
func (s *HTTPServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	nonce := nonceFromContext(r.Context())
	body := bytes.ReplaceAll(
		s.indexHTML,
		[]byte("<script"),
		fmt.Appendf(nil, `<script nonce="%s"`, nonce),
	)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(body)
}

// Wraps an http.Handler to add Cache-Control headers
func cacheControlMiddleware(next http.Handler, cacheControl string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", cacheControl)
		next.ServeHTTP(w, r)
	})
}

// Pre-computes the /api/config JSON response and ETag.
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

// Starts the HTTP server
func (s *HTTPServer) Start() error {
	log.Printf("HTTP server starting on %s:%d", s.host, s.port)
	return s.server.ListenAndServe()
}

// Gracefully shuts down the HTTP server
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Starts the server and handles shutdown signals
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
