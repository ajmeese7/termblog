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

	host       string
	port       int
	binaryPath string
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(host string, port int, repo *storage.PostRepository, loader *blog.ContentLoader, feed *blog.FeedGenerator, binaryPath string) *HTTPServer {
	s := &HTTPServer{
		repo:       repo,
		loader:     loader,
		feed:       feed,
		host:       host,
		port:       port,
		binaryPath: binaryPath,
	}

	mux := http.NewServeMux()

	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Routes
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/feed.xml", s.handleRSSFeed)
	mux.HandleFunc("/feed.json", s.handleJSONFeed)
	mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// handleIndex serves the main terminal page
func (s *HTTPServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tmpl, err := template.ParseFS(templatesFS, "templates/index.html")
	if err != nil {
		log.Printf("Failed to parse template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := struct {
		Title string
		WSUrl string
	}{
		Title: "Terminal Blog",
		WSUrl: fmt.Sprintf("ws://%s/ws", r.Host),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
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
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

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
