package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ajmeese7/termblog/internal/app"
	"github.com/ajmeese7/termblog/internal/blog"
	"github.com/ajmeese7/termblog/internal/server"
	"github.com/ajmeese7/termblog/internal/storage"
	"github.com/ajmeese7/termblog/internal/theme"
	"github.com/ajmeese7/termblog/internal/tui"
	"github.com/ajmeese7/termblog/internal/version"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "termblog",
		Short: "A terminal-based blog platform",
		Long:  "Termblog is a self-hosted TUI blog platform accessible via SSH or web browser.",
	}

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "config.yaml", "config file path")

	rootCmd.AddCommand(serveCmd())
	rootCmd.AddCommand(newCmd())
	rootCmd.AddCommand(ptyCmd())
	rootCmd.AddCommand(syncCmd())
	rootCmd.AddCommand(publishCmd())
	rootCmd.AddCommand(versionCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func serveCmd() *cobra.Command {
	var sshOnly, httpOnly bool

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the blog server",
		Long:  "Start both SSH and HTTP servers for the terminal blog.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(sshOnly, httpOnly)
		},
	}

	cmd.Flags().BoolVar(&sshOnly, "ssh-only", false, "Only start the SSH server")
	cmd.Flags().BoolVar(&httpOnly, "http-only", false, "Only start the HTTP server")

	return cmd
}

func newCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new [title]",
		Short: "Create a new blog post",
		Long:  "Create a new blog post with the given title.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNew(args[0])
		},
	}
}

func ptyCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "pty",
		Short:  "Run the TUI in PTY mode (used by web terminal)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPTY()
		},
	}
}

func syncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync markdown files to database",
		Long:  "Scan the content directory and sync posts to the database.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSync()
		},
	}
}

func publishCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "publish <slug>",
		Short: "Publish a draft post",
		Long:  "Publish a draft post by setting its status to published and updating the frontmatter.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPublish(args[0])
		},
	}
}

func versionCmd() *cobra.Command {
	var full bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			if full {
				fmt.Println(version.Full())
			} else {
				fmt.Println(version.Info())
			}
		},
	}

	cmd.Flags().BoolVarP(&full, "full", "f", false, "Show full version info (commit, date)")

	return cmd
}

func runServe(sshOnly, httpOnly bool) error {
	cfg, err := app.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	appInstance, err := app.New(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	// Open database
	db, err := storage.Open(appInstance.DatabasePath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	repo := storage.NewPostRepository(db)
	loader := blog.NewContentLoader(appInstance.ContentPath())
	t := theme.GetTheme(cfg.Theme, "")

	// Sync posts on startup
	if err := syncPosts(loader, repo); err != nil {
		log.Printf("Warning: failed to sync posts: %v", err)
	}

	tuiConfig := tui.Config{
		BlogTitle: cfg.Blog.Title,
		Author:    cfg.Blog.Author,
	}

	// Get the binary path for PTY spawning
	binaryPath, err := os.Executable()
	if err != nil {
		binaryPath = os.Args[0]
	}

	// Create feed generator
	feedGen := blog.NewFeedGenerator(
		cfg.Blog.Title,
		cfg.Blog.Description,
		cfg.Blog.Author,
		cfg.Blog.BaseURL,
	)

	// Setup signal handling
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	// Start servers
	errCh := make(chan error, 2)

	if !httpOnly {
		sshServer, err := server.NewSSHServer(
			"0.0.0.0",
			cfg.Server.SSHPort,
			appInstance.HostKeyPath(),
			repo,
			loader,
			t,
			tuiConfig,
		)
		if err != nil {
			return fmt.Errorf("failed to create SSH server: %w", err)
		}

		go func() {
			log.Printf("SSH server listening on :%d", cfg.Server.SSHPort)
			if err := sshServer.Start(); err != nil {
				errCh <- fmt.Errorf("SSH server error: %w", err)
			}
		}()

		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			sshServer.Shutdown(ctx)
		}()
	}

	if !sshOnly {
		httpServer := server.NewHTTPServer(
			"0.0.0.0",
			cfg.Server.HTTPPort,
			repo,
			loader,
			feedGen,
			binaryPath,
		)

		go func() {
			log.Printf("HTTP server listening on :%d", cfg.Server.HTTPPort)
			if err := httpServer.Start(); err != nil {
				errCh <- fmt.Errorf("HTTP server error: %w", err)
			}
		}()

		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			httpServer.Shutdown(ctx)
		}()
	}

	log.Println("Termblog is running. Press Ctrl+C to stop.")

	select {
	case <-done:
		log.Println("Shutting down...")
	case err := <-errCh:
		return err
	}

	return nil
}

func runNew(title string) error {
	cfg, err := app.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	appInstance, err := app.New(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	loader := blog.NewContentLoader(appInstance.ContentPath())
	filePath, err := loader.CreatePost(title, cfg.Blog.Author)
	if err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}

	fmt.Printf("Created new post: %s\n", filePath)
	fmt.Printf("Edit the file and set 'draft: false' to publish.\n")

	return nil
}

func runPTY() error {
	cfg, err := app.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	appInstance, err := app.New(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	// Open database
	db, err := storage.Open(appInstance.DatabasePath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	repo := storage.NewPostRepository(db)
	loader := blog.NewContentLoader(appInstance.ContentPath())
	t := theme.GetTheme(cfg.Theme, "")

	tuiConfig := tui.Config{
		BlogTitle: cfg.Blog.Title,
		Author:    cfg.Blog.Author,
	}

	model := tui.New(repo, loader, t, tuiConfig)

	// Build program options
	opts := []tea.ProgramOption{
		tea.WithAltScreen(),
	}

	// Only enable mouse if not disabled via environment
	// Web terminal sets TERMBLOG_NO_MOUSE=1 to allow text selection
	if os.Getenv("TERMBLOG_NO_MOUSE") == "" {
		opts = append(opts, tea.WithMouseCellMotion())
	}

	p := tea.NewProgram(model, opts...)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

func runSync() error {
	appInstance, err := app.New(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	// Open database
	db, err := storage.Open(appInstance.DatabasePath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	repo := storage.NewPostRepository(db)
	loader := blog.NewContentLoader(appInstance.ContentPath())

	count, err := syncPostsWithCount(loader, repo)
	if err != nil {
		return fmt.Errorf("failed to sync posts: %w", err)
	}

	fmt.Printf("Synced %d posts to database.\n", count)
	return nil
}

func runPublish(slug string) error {
	appInstance, err := app.New(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	// Open database
	db, err := storage.Open(appInstance.DatabasePath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	repo := storage.NewPostRepository(db)

	// Find the post
	post, err := repo.GetBySlug(slug)
	if err != nil {
		return fmt.Errorf("failed to find post: %w", err)
	}
	if post == nil {
		return fmt.Errorf("post not found: %s", slug)
	}

	if post.Status == storage.StatusPublished {
		fmt.Printf("Post '%s' is already published.\n", post.Title)
		return nil
	}

	// Update database
	now := time.Now()
	post.Status = storage.StatusPublished
	post.PublishedAt = &now

	if err := repo.Update(post); err != nil {
		return fmt.Errorf("failed to update post in database: %w", err)
	}

	fmt.Printf("Published: %s\n", post.Title)
	return nil
}

func syncPosts(loader *blog.ContentLoader, repo *storage.PostRepository) error {
	_, err := syncPostsWithCount(loader, repo)
	return err
}

func syncPostsWithCount(loader *blog.ContentLoader, repo *storage.PostRepository) (int, error) {
	posts, err := loader.LoadAllPosts()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, post := range posts {
		status := storage.StatusDraft
		if !post.Draft {
			status = storage.StatusPublished
		}

		absPath, err := filepath.Abs(post.Filepath)
		if err != nil {
			absPath = post.Filepath
		}

		dbPost := &storage.Post{
			Slug:        post.Slug,
			Title:       post.Title,
			Filepath:    absPath,
			Status:      status,
			Tags:        post.Tags,
			PublishedAt: post.PublishedAt,
		}

		if err := repo.UpsertBySlug(dbPost); err != nil {
			log.Printf("Warning: failed to sync post %s: %v", post.Slug, err)
			continue
		}
		count++
	}

	return count, nil
}
