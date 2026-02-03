package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ajmeese7/termblog/internal/blog"
	"github.com/ajmeese7/termblog/internal/storage"
	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static
var staticFS embed.FS

// HTTPServer handles HTTP requests and WebSocket connections
type HTTPServer struct {
	server *http.Server
	repo   *storage.PostRepository
	loader *blog.ContentLoader
	feed   *blog.FeedGenerator

	host            string
	port            int
	binaryPath      string
	blogTitle       string
	blogDescription string

	// Cached templates (parsed once at startup)
	templates map[string]*template.Template
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(host string, port int, repo *storage.PostRepository, loader *blog.ContentLoader, feed *blog.FeedGenerator, binaryPath string, blogTitle string, blogDescription string) (*HTTPServer, error) {
	// Parse and cache templates at startup for efficiency and early error detection
	templates := make(map[string]*template.Template)
	templateNames := []string{"index.html", "archive.html", "post.html", "tag.html"}

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
		loader:          loader,
		feed:            feed,
		host:            host,
		port:            port,
		binaryPath:      binaryPath,
		blogTitle:       blogTitle,
		blogDescription: blogDescription,
		templates:       templates,
	}

	mux := http.NewServeMux()

	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Routes
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/feed.xml", s.handleRSSFeed)
	mux.HandleFunc("/feed.json", s.handleJSONFeed)
	mux.HandleFunc("/sitemap.xml", s.handleSitemap)
	mux.HandleFunc("/robots.txt", s.handleRobots)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/archive", s.handleArchive)
	mux.HandleFunc("/posts/", s.handlePost)
	mux.HandleFunc("/tags/", s.handleTag)

	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s, nil
}

// handleIndex serves the main terminal page
func (s *HTTPServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := struct {
		Title string
		WSUrl string
	}{
		Title: s.blogTitle,
		WSUrl: fmt.Sprintf("ws://%s/ws", r.Host),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["index.html"].Execute(w, data); err != nil {
		log.Printf("Failed to execute template: %v", err)
	}
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
		Posts           []*blog.Post
	}{
		BlogTitle:       s.blogTitle,
		BlogDescription: s.blogDescription,
		Posts:           blogPosts,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
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
	}{
		BlogTitle: s.blogTitle,
		BaseURL:   s.feed.BaseURL(),
		Post:      post,
		Content:   template.HTML(htmlContent),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
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
		Tag       string
		Posts     []*blog.Post
	}{
		BlogTitle: s.blogTitle,
		Tag:       tag,
		Posts:     blogPosts,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates["tag.html"].Execute(w, data); err != nil {
		log.Printf("Failed to execute tag template: %v", err)
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// handleWebSocket handles WebSocket connections for the PTY bridge
func (s *HTTPServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Spawn the PTY process
	cmd := exec.Command(s.binaryPath, "pty")
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"TERMBLOG_NO_MOUSE=1", // Disable mouse mode to allow text selection in browser
	)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Printf("Failed to start PTY: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: %v", err)))
		return
	}
	defer ptmx.Close()

	// Set initial size
	pty.Setsize(ptmx, &pty.Winsize{
		Rows: 24,
		Cols: 80,
	})

	var wg sync.WaitGroup
	done := make(chan struct{})

	// PTY -> WebSocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			select {
			case <-done:
				return
			default:
				n, err := ptmx.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("PTY read error: %v", err)
					}
					return
				}
				if n > 0 {
					if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
						log.Printf("WebSocket write error: %v", err)
						return
					}
				}
			}
		}
	}()

	// WebSocket -> PTY
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(done)
		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket read error: %v", err)
				}
				return
			}

			switch msgType {
			case websocket.BinaryMessage, websocket.TextMessage:
				// Check for resize message
				if len(msg) > 0 && msg[0] == '\x01' {
					// Resize message format: \x01{rows},{cols}
					var rows, cols uint16
					if _, err := fmt.Sscanf(string(msg[1:]), "%d,%d", &rows, &cols); err == nil {
						pty.Setsize(ptmx, &pty.Winsize{
							Rows: rows,
							Cols: cols,
						})
					}
				} else {
					// Regular input
					if _, err := ptmx.Write(msg); err != nil {
						log.Printf("PTY write error: %v", err)
						return
					}
				}
			}
		}
	}()

	// Wait for process to exit
	cmd.Wait()
	wg.Wait()
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
