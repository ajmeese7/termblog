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
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/muesli/termenv"
	gossh "golang.org/x/crypto/ssh"
)

// SSHServer wraps the Wish SSH server
type SSHServer struct {
	server      *ssh.Server
	repo        *storage.PostRepository
	prefRepo    *storage.PreferenceRepository
	viewRepo    *storage.ViewRepository
	loader      *blog.ContentLoader
	theme       *theme.Theme
	config      tui.Config
	rateLimiter *RateLimiter
	cmdHandler  *CommandHandler

	host              string
	port              int
	adminFingerprints []string
}

// SSHConfig holds SSH server-specific configuration
type SSHConfig struct {
	RateLimitCount    int
	RateLimitWindow   time.Duration
	ExitMessage       string
	FeedGenerator     *blog.FeedGenerator // For RSS command support
	AdminFingerprints []string            // SSH fingerprints with admin access
}

// NewSSHServer creates a new SSH server
func NewSSHServer(host string, port int, hostKeyPath string, repo *storage.PostRepository, prefRepo *storage.PreferenceRepository, viewRepo *storage.ViewRepository, loader *blog.ContentLoader, t *theme.Theme, tuiCfg tui.Config, sshCfg SSHConfig) (*SSHServer, error) {
	// Create rate limiter with configurable settings
	rateLimiter := NewRateLimiter(sshCfg.RateLimitCount, sshCfg.RateLimitWindow)

	// Create command handler for non-interactive SSH commands
	cmdHandler := NewCommandHandler(repo, loader, sshCfg.FeedGenerator)

	s := &SSHServer{
		repo:              repo,
		prefRepo:          prefRepo,
		viewRepo:          viewRepo,
		loader:            loader,
		theme:             t,
		config:            tuiCfg,
		rateLimiter:       rateLimiter,
		cmdHandler:        cmdHandler,
		host:              host,
		port:              port,
		adminFingerprints: sshCfg.AdminFingerprints,
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
			s.commandMiddleware(), // Handle non-interactive commands
			bubbletea.MiddlewareWithColorProfile(s.teaHandler, termenv.TrueColor),
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
	// Get SSH key fingerprint for theme persistence and admin check
	fingerprint := ""
	if pubKey := sshSession.PublicKey(); pubKey != nil {
		fingerprint = gossh.FingerprintSHA256(pubKey)
	}

	// Check if user is admin
	isAdmin := s.isAdminFingerprint(fingerprint)
	if isAdmin {
		log.Printf("Admin session started for fingerprint: %s", fingerprint)
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

	renderer := bubbletea.MakeRenderer(sshSession)
	model := tui.NewWithPreferences(s.repo, s.loader, selectedTheme, s.config, fingerprint, s.prefRepo, s.viewRepo, isAdmin, renderer)

	return model, []tea.ProgramOption{
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	}
}

// isAdminFingerprint checks if the given fingerprint is in the admin whitelist
func (s *SSHServer) isAdminFingerprint(fingerprint string) bool {
	if fingerprint == "" {
		return false
	}
	for _, fp := range s.adminFingerprints {
		if fp == fingerprint {
			return true
		}
	}
	return false
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
			// Skip for non-interactive command sessions
			if msg := strings.TrimSpace(message); msg != "" && len(sess.Command()) == 0 {
				wish.Println(sess, "")
				wish.Println(sess, msg)
				wish.Println(sess, "")
			}
		}
	}
}

// commandMiddleware handles non-interactive SSH commands
// If a command is provided, it's handled directly without launching the TUI
func (s *SSHServer) commandMiddleware() wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(sess ssh.Session) {
			cmd := sess.Command()

			// If no command, proceed to TUI
			if len(cmd) == 0 {
				next(sess)
				return
			}

			// Handle the command
			handled, err := s.cmdHandler.HandleCommand(sess, cmd)
			if err != nil {
				wish.Fatalln(sess, fmt.Sprintf("Error: %v", err))
				return
			}

			// If command wasn't handled, check if we can fall through to TUI
			if !handled {
				// Non-PTY sessions can't run the TUI — show an error
				if _, _, ok := sess.Pty(); !ok {
					wish.Fatalln(sess, fmt.Sprintf("Unknown command: %s\nRun 'help' for available commands.", cmd[0]))
					return
				}
				next(sess)
				return
			}

			// Command was handled, session will close
		}
	}
}
