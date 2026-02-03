package app

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	Blog    BlogConfig    `yaml:"blog"`
	Server  ServerConfig  `yaml:"server"`
	Storage StorageConfig `yaml:"storage"`
	Theme   string        `yaml:"theme"`
}

// BlogConfig holds blog-specific settings
type BlogConfig struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Author      string `yaml:"author"`
	BaseURL     string `yaml:"base_url"`
	ContentDir  string `yaml:"content_dir"`
	ExitMessage string `yaml:"exit_message"`
	MOTD        string `yaml:"motd"` // Message of the day shown on SSH connect
}

// ServerConfig holds server settings
type ServerConfig struct {
	SSHPort     int             `yaml:"ssh_port"`
	HTTPPort    int             `yaml:"http_port"`
	HostKeyPath string          `yaml:"host_key_path"`
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
	}
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
