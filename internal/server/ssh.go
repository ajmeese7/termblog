package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ajmeese7/termblog/internal/blog"
	"github.com/ajmeese7/termblog/internal/storage"
	"github.com/ajmeese7/termblog/internal/theme"
	"github.com/ajmeese7/termblog/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
)

// SSHServer wraps the Wish SSH server
type SSHServer struct {
	server *ssh.Server
	repo   *storage.PostRepository
	loader *blog.ContentLoader
	theme  *theme.Theme
	config tui.Config

	host string
	port int
}

// NewSSHServer creates a new SSH server
func NewSSHServer(host string, port int, hostKeyPath string, repo *storage.PostRepository, loader *blog.ContentLoader, t *theme.Theme, cfg tui.Config) (*SSHServer, error) {
	s := &SSHServer{
		repo:   repo,
		loader: loader,
		theme:  t,
		config: cfg,
		host:   host,
		port:   port,
	}

	// Ensure host key directory exists
	if err := ensureHostKey(hostKeyPath); err != nil {
		return nil, fmt.Errorf("failed to ensure host key: %w", err)
	}

	server, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
		wish.WithHostKeyPath(hostKeyPath),
		wish.WithMiddleware(
			bubbletea.Middleware(s.teaHandler),
			activeterm.Middleware(),
			logging.Middleware(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH server: %w", err)
	}

	s.server = server
	return s, nil
}

// teaHandler returns a new Bubbletea program for each SSH session
func (s *SSHServer) teaHandler(sshSession ssh.Session) (tea.Model, []tea.ProgramOption) {
	model := tui.New(s.repo, s.loader, s.theme, s.config)

	return model, []tea.ProgramOption{
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	}
}

// Start starts the SSH server
func (s *SSHServer) Start() error {
	log.Printf("SSH server starting on %s:%d", s.host, s.port)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the SSH server
func (s *SSHServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// ListenAndServeWithSignal starts the server and handles shutdown signals
func (s *SSHServer) ListenAndServeWithSignal() error {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	log.Printf("SSH server starting on %s:%d", s.host, s.port)

	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			log.Printf("SSH server error: %v", err)
		}
	}()

	<-done
	log.Println("SSH server shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

// ensureHostKey ensures the host key directory exists
func ensureHostKey(path string) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	return nil
}
