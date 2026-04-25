package app

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	Blog    BlogConfig    `yaml:"blog"`
	Server  ServerConfig  `yaml:"server"`
	Storage StorageConfig `yaml:"storage"`
	Admin   AdminConfig   `yaml:"admin"`
	Theme   string        `yaml:"theme"`
	Favicon FaviconConfig `yaml:"favicon"`
}

// Favicon modes
const (
	FaviconModeLetter = "letter"
	FaviconModeEmoji  = "emoji"
	FaviconModeImage  = "image"
)

// Emoji background modes
const (
	FaviconEmojiBgTransparent = "transparent"
	FaviconEmojiBgThemed      = "themed"
)

// FaviconConfig controls dynamic favicon rendering. The feature ships enabled
// by default in letter mode; set Enabled to false to fall back to the static
// /static/favicon.ico (if present).
type FaviconConfig struct {
	Enabled bool   `yaml:"enabled"`
	Mode    string `yaml:"mode"`     // "letter" | "emoji" | "image"
	Letter  string `yaml:"letter"`   // single rune for letter mode
	Emoji   string `yaml:"emoji"`    // glyph for emoji mode
	EmojiBg string `yaml:"emoji_bg"` // "transparent" | "themed"
	Image   string `yaml:"image"`    // local path or http(s):// URL for image mode

	// ResolvedImagePath is the absolute path to the image file when Mode is
	// "image" and Image is a local path. Set during validation; empty for URLs.
	ResolvedImagePath string `yaml:"-"`
}

// AdminConfig holds admin authentication settings
type AdminConfig struct {
	// Fingerprints is a list of SSH key fingerprints allowed to access admin features
	// Format: SHA256:... (as shown by `ssh-keygen -lf ~/.ssh/id_ed25519.pub`)
	Fingerprints []string `yaml:"fingerprints"`
}

// BlogConfig holds blog-specific settings
type BlogConfig struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Author      string `yaml:"author"`
	BaseURL     string `yaml:"base_url"`
	ContentDir  string `yaml:"content_dir"`
	ExitMessage string `yaml:"exit_message"`
	ASCIIHeader string `yaml:"ascii_header"` // Optional ASCII art header file path
}

// ServerConfig holds server settings
type ServerConfig struct {
	SSHPort     int             `yaml:"ssh_port"`
	HTTPPort    int             `yaml:"http_port"`
	HostKeyPath string          `yaml:"host_key_path"`
	TrustProxy  bool            `yaml:"trust_proxy"`
	RateLimit   RateLimitConfig `yaml:"rate_limit"`
}

// RateLimitConfig holds rate limiting settings
type RateLimitConfig struct {
	Limit  int `yaml:"limit"`  // Max connections per window
	Window int `yaml:"window"` // Window duration in seconds
}

// StorageConfig holds storage settings
type StorageConfig struct {
	DatabasePath string `yaml:"database_path"`
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Blog: BlogConfig{
			Title:       "My Terminal Blog",
			Description: "A blog you can read in your terminal",
			Author:      "Anonymous",
			BaseURL:     "http://localhost:8080",
			ContentDir:  "content/posts",
		},
		Server: ServerConfig{
			SSHPort:     2222,
			HTTPPort:    8080,
			HostKeyPath: ".ssh/termblog_host_key",
			RateLimit: RateLimitConfig{
				Limit:  10, // 10 connections per minute
				Window: 60, // 60 seconds
			},
		},
		Storage: StorageConfig{
			DatabasePath: "termblog.db",
		},
		Theme: "pipboy",
		Favicon: FaviconConfig{
			Enabled: true,
			Mode:    FaviconModeLetter,
			Letter:  "T",
			Emoji:   "📝",
			EmojiBg: FaviconEmojiBgTransparent,
		},
	}
}

// allowedImageExts is the set of extensions accepted for local image-mode
// favicons. Anything outside this set is rejected at config load time.
var allowedImageExts = map[string]struct{}{
	".svg":  {},
	".png":  {},
	".ico":  {},
	".jpg":  {},
	".jpeg": {},
	".webp": {},
	".gif":  {},
}

// validateAndResolveFavicon normalizes the favicon section and returns a
// helpful error if the operator set something nonsensical. It mutates cfg in
// place so downstream code can rely on resolved fields.
//
// configDir is used to resolve relative image paths.
func validateAndResolveFavicon(cfg *FaviconConfig, configDir string) error {
	if !cfg.Enabled {
		return nil
	}

	switch cfg.Mode {
	case "":
		cfg.Mode = FaviconModeLetter
	case FaviconModeLetter, FaviconModeEmoji, FaviconModeImage:
	default:
		return fmt.Errorf("favicon.mode %q invalid (want letter, emoji, or image)", cfg.Mode)
	}

	if cfg.Letter == "" {
		cfg.Letter = "T"
	}
	if cfg.Emoji == "" {
		cfg.Emoji = "📝"
	}
	switch cfg.EmojiBg {
	case "":
		cfg.EmojiBg = FaviconEmojiBgTransparent
	case FaviconEmojiBgTransparent, FaviconEmojiBgThemed:
	default:
		return fmt.Errorf("favicon.emoji_bg %q invalid (want transparent or themed)", cfg.EmojiBg)
	}

	if cfg.Mode != FaviconModeImage {
		return nil
	}

	if strings.TrimSpace(cfg.Image) == "" {
		return fmt.Errorf("favicon.mode is image but favicon.image is empty")
	}

	// URL form: only http(s) is allowed. file://, data:, javascript:, etc. are
	// rejected to avoid SSRF-flavored misconfigurations and surprises.
	if strings.HasPrefix(cfg.Image, "http://") || strings.HasPrefix(cfg.Image, "https://") {
		u, err := url.Parse(cfg.Image)
		if err != nil {
			return fmt.Errorf("favicon.image is not a valid URL: %w", err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("favicon.image scheme %q not allowed (want http or https)", u.Scheme)
		}
		if u.Host == "" {
			return fmt.Errorf("favicon.image URL missing host")
		}
		return nil
	}

	// Local path form: resolve, verify, and lock down extension.
	path := cfg.Image
	if !filepath.IsAbs(path) {
		path = filepath.Join(configDir, path)
	}
	clean := filepath.Clean(path)

	info, err := os.Stat(clean)
	if err != nil {
		return fmt.Errorf("favicon.image %q not readable: %w", cfg.Image, err)
	}
	if info.IsDir() {
		return fmt.Errorf("favicon.image %q is a directory", cfg.Image)
	}

	ext := strings.ToLower(filepath.Ext(clean))
	if _, ok := allowedImageExts[ext]; !ok {
		return fmt.Errorf("favicon.image extension %q not allowed", ext)
	}

	cfg.ResolvedImagePath = clean
	return nil
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	configDir := filepath.Dir(path)
	if err := validateAndResolveFavicon(&cfg.Favicon, configDir); err != nil {
		return nil, fmt.Errorf("invalid favicon config: %w", err)
	}

	return cfg, nil
}

// App holds the shared application state
type App struct {
	Config *Config
	Root   string // Root directory of the application
}

// New creates a new App instance
func New(configPath string) (*App, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	root, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	return &App{
		Config: cfg,
		Root:   root,
	}, nil
}

// ContentPath returns the full path to the content directory
func (a *App) ContentPath() string {
	if filepath.IsAbs(a.Config.Blog.ContentDir) {
		return a.Config.Blog.ContentDir
	}
	return filepath.Join(a.Root, a.Config.Blog.ContentDir)
}

// DatabasePath returns the full path to the database
func (a *App) DatabasePath() string {
	if filepath.IsAbs(a.Config.Storage.DatabasePath) {
		return a.Config.Storage.DatabasePath
	}
	return filepath.Join(a.Root, a.Config.Storage.DatabasePath)
}

// HostKeyPath returns the full path to the SSH host key
func (a *App) HostKeyPath() string {
	if filepath.IsAbs(a.Config.Server.HostKeyPath) {
		return a.Config.Server.HostKeyPath
	}
	return filepath.Join(a.Root, a.Config.Server.HostKeyPath)
}

// IsAdmin checks if the given SSH fingerprint is in the admin whitelist
func (a *App) IsAdmin(fingerprint string) bool {
	for _, fp := range a.Config.Admin.Fingerprints {
		if fp == fingerprint {
			return true
		}
	}
	return false
}
