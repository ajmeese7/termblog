package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
	gossh "golang.org/x/crypto/ssh"
)

// SSHServer wraps the Wish SSH server
type SSHServer struct {
	server      *ssh.Server
	repo        *storage.PostRepository
	prefRepo    *storage.PreferenceRepository
	loader      *blog.ContentLoader
	theme       *theme.Theme
	config      tui.Config
	rateLimiter *RateLimiter

	host string
	port int
}

// SSHConfig holds SSH server-specific configuration
type SSHConfig struct {
	RateLimitCount  int
	RateLimitWindow time.Duration
	ExitMessage     string
}

// NewSSHServer creates a new SSH server
func NewSSHServer(host string, port int, hostKeyPath string, repo *storage.PostRepository, prefRepo *storage.PreferenceRepository, loader *blog.ContentLoader, t *theme.Theme, tuiCfg tui.Config, sshCfg SSHConfig) (*SSHServer, error) {
	// Create rate limiter with configurable settings
	rateLimiter := NewRateLimiter(sshCfg.RateLimitCount, sshCfg.RateLimitWindow)

	s := &SSHServer{
		repo:        repo,
		prefRepo:    prefRepo,
		loader:      loader,
		theme:       t,
		config:      tuiCfg,
		rateLimiter: rateLimiter,
		host:        host,
		port:        port,
	}

	// Ensure host key directory exists
	if err := ensureHostKey(hostKeyPath); err != nil {
		return nil, fmt.Errorf("failed to ensure host key: %w", err)
	}

	server, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
		wish.WithHostKeyPath(hostKeyPath),
		wish.WithPublicKeyAuth(func(_ ssh.Context, _ ssh.PublicKey) bool {
			// Accept all public keys - we just need the fingerprint for theme persistence
			return true
		}),
		wish.WithMiddleware(
			exitMessageMiddleware(sshCfg.ExitMessage),
			bubbletea.Middleware(s.teaHandler),
			activeterm.Middleware(),
			RateLimitMiddleware(rateLimiter),
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
	// Get SSH key fingerprint for theme persistence
	fingerprint := ""
	if pubKey := sshSession.PublicKey(); pubKey != nil {
		fingerprint = gossh.FingerprintSHA256(pubKey)
	}

	// Load user's preferred theme
	selectedTheme := s.theme
	if fingerprint != "" && s.prefRepo != nil {
		if themeName, err := s.prefRepo.GetTheme(fingerprint); err == nil {
			if t := theme.GetTheme(themeName, ""); t != nil {
				selectedTheme = t
			}
		}
	}

	model := tui.NewWithPreferences(s.repo, s.loader, selectedTheme, s.config, fingerprint, s.prefRepo)

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
	if s.rateLimiter != nil {
		s.rateLimiter.Stop()
	}
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

// exitMessageMiddleware displays a message after the TUI exits
func exitMessageMiddleware(message string) wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			next(sess)
			// Print exit message after TUI closes (if configured)
			if msg := strings.TrimSpace(message); msg != "" {
				wish.Println(sess, "")
				wish.Println(sess, msg)
				wish.Println(sess, "")
			}
		}
	}
}
