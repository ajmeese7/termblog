package main

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
	rootCmd.AddCommand(unpublishCmd())
	rootCmd.AddCommand(deleteCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(scheduleCmd())
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

func unpublishCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unpublish <slug>",
		Short: "Unpublish a post",
		Long:  "Revert a published post to draft status.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUnpublish(args[0])
		},
	}
}

func deleteCmd() *cobra.Command {
	var removeFile bool

	cmd := &cobra.Command{
		Use:   "delete <slug>",
		Short: "Delete a post",
		Long:  "Delete a post from the database and optionally from the filesystem.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(args[0], removeFile)
		},
	}

	cmd.Flags().BoolVarP(&removeFile, "remove-file", "r", false, "Also delete the markdown file")

	return cmd
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all posts",
		Long:  "List all posts with their status (draft/published/scheduled).",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList()
		},
	}
}

func scheduleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "schedule <slug> <datetime>",
		Short: "Schedule a post for publication",
		Long:  "Schedule a post for future publication. Datetime format: YYYY-MM-DD or YYYY-MM-DDTHH:MM:SS",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSchedule(args[0], args[1])
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
	prefRepo := storage.NewPreferenceRepository(db)
	viewRepo := storage.NewViewRepository(db)
	loader := blog.NewContentLoader(appInstance.ContentPath())
	t := theme.GetTheme(cfg.Theme, "")

	// Sync posts on startup
	if err := syncPosts(loader, repo); err != nil {
		log.Printf("Warning: failed to sync posts: %v", err)
	}

	// Start file watcher for auto-sync
	watcher, err := storage.NewContentWatcher(appInstance.ContentPath(), func() error {
		return syncPosts(loader, repo)
	})
	if err != nil {
		log.Printf("Warning: failed to create file watcher: %v", err)
	} else {
		if err := watcher.Start(); err != nil {
			log.Printf("Warning: failed to start file watcher: %v", err)
		} else {
			defer watcher.Stop()
		}
	}

	// Load optional ASCII header
	var asciiHeader string
	if cfg.Blog.ASCIIHeader != "" {
		headerPath := cfg.Blog.ASCIIHeader
		if !filepath.IsAbs(headerPath) {
			headerPath = filepath.Join(appInstance.Root, headerPath)
		}
		if data, err := os.ReadFile(headerPath); err == nil {
			asciiHeader = strings.TrimSpace(string(data))
		} else {
			log.Printf("Warning: failed to load ASCII header: %v", err)
		}
	}

	tuiConfig := tui.Config{
		BlogTitle:   cfg.Blog.Title,
		Author:      cfg.Blog.Author,
		ASCIIHeader: asciiHeader,
		ContentDir:  appInstance.ContentPath(),
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
		sshCfg := server.SSHConfig{
			RateLimitCount:    cfg.Server.RateLimit.Limit,
			RateLimitWindow:   time.Duration(cfg.Server.RateLimit.Window) * time.Second,
			ExitMessage:       cfg.Blog.ExitMessage,
			FeedGenerator:     feedGen,
			AdminFingerprints: cfg.Admin.Fingerprints,
		}
		sshServer, err := server.NewSSHServer(
			"0.0.0.0",
			cfg.Server.SSHPort,
			appInstance.HostKeyPath(),
			repo,
			prefRepo,
			viewRepo,
			loader,
			t,
			tuiConfig,
			sshCfg,
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
		httpServer, err := server.NewHTTPServer(
			"0.0.0.0",
			cfg.Server.HTTPPort,
			repo,
			loader,
			feedGen,
			binaryPath,
			cfg.Blog.Title,
			cfg.Blog.Description,
			t,
			cfg.Theme,
		)
		if err != nil {
			return fmt.Errorf("failed to create HTTP server: %w", err)
		}

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

	// Load optional ASCII header
	var asciiHeader string
	if cfg.Blog.ASCIIHeader != "" {
		headerPath := cfg.Blog.ASCIIHeader
		if !filepath.IsAbs(headerPath) {
			headerPath = filepath.Join(appInstance.Root, headerPath)
		}
		if data, err := os.ReadFile(headerPath); err == nil {
			asciiHeader = strings.TrimSpace(string(data))
		}
	}

	tuiConfig := tui.Config{
		BlogTitle:   cfg.Blog.Title,
		Author:      cfg.Blog.Author,
		ASCIIHeader: asciiHeader,
		ContentDir:  appInstance.ContentPath(),
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

func runUnpublish(slug string) error {
	appInstance, err := app.New(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	db, err := storage.Open(appInstance.DatabasePath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	repo := storage.NewPostRepository(db)

	post, err := repo.GetBySlug(slug)
	if err != nil {
		return fmt.Errorf("failed to find post: %w", err)
	}
	if post == nil {
		return fmt.Errorf("post not found: %s", slug)
	}

	if post.Status == storage.StatusDraft {
		fmt.Printf("Post '%s' is already a draft.\n", post.Title)
		return nil
	}

	post.Status = storage.StatusDraft
	post.PublishedAt = nil

	if err := repo.Update(post); err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	fmt.Printf("Unpublished: %s (now a draft)\n", post.Title)
	return nil
}

func runDelete(slug string, removeFile bool) error {
	appInstance, err := app.New(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	db, err := storage.Open(appInstance.DatabasePath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	repo := storage.NewPostRepository(db)

	post, err := repo.GetBySlug(slug)
	if err != nil {
		return fmt.Errorf("failed to find post: %w", err)
	}
	if post == nil {
		return fmt.Errorf("post not found: %s", slug)
	}

	// Delete from database
	if err := repo.Delete(post.ID); err != nil {
		return fmt.Errorf("failed to delete post from database: %w", err)
	}

	fmt.Printf("Deleted from database: %s\n", post.Title)

	// Optionally delete the file
	if removeFile && post.Filepath != "" {
		if err := os.Remove(post.Filepath); err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("File already deleted: %s\n", post.Filepath)
			} else {
				return fmt.Errorf("failed to delete file: %w", err)
			}
		} else {
			fmt.Printf("Deleted file: %s\n", post.Filepath)
		}
	}

	return nil
}

func runList() error {
	appInstance, err := app.New(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	db, err := storage.Open(appInstance.DatabasePath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	repo := storage.NewPostRepository(db)

	posts, err := repo.ListAll(1000, 0)
	if err != nil {
		return fmt.Errorf("failed to list posts: %w", err)
	}

	if len(posts) == 0 {
		fmt.Println("No posts found.")
		return nil
	}

	// Print header
	fmt.Printf("%-12s  %-20s  %s\n", "STATUS", "SLUG", "TITLE")
	fmt.Println(strings.Repeat("-", 60))

	for _, post := range posts {
		status := string(post.Status)
		if post.Status == storage.StatusScheduled && post.PublishedAt != nil {
			status = fmt.Sprintf("scheduled (%s)", post.PublishedAt.Format("2006-01-02"))
		}
		fmt.Printf("%-12s  %-20s  %s\n", status, post.Slug, post.Title)
	}

	return nil
}

func runSchedule(slug, datetime string) error {
	appInstance, err := app.New(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	db, err := storage.Open(appInstance.DatabasePath())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	repo := storage.NewPostRepository(db)

	post, err := repo.GetBySlug(slug)
	if err != nil {
		return fmt.Errorf("failed to find post: %w", err)
	}
	if post == nil {
		return fmt.Errorf("post not found: %s", slug)
	}

	// Parse datetime (supports YYYY-MM-DD or YYYY-MM-DDTHH:MM:SS)
	var publishAt time.Time
	if len(datetime) == 10 {
		publishAt, err = time.Parse("2006-01-02", datetime)
	} else {
		publishAt, err = time.Parse("2006-01-02T15:04:05", datetime)
	}
	if err != nil {
		return fmt.Errorf("invalid datetime format (use YYYY-MM-DD or YYYY-MM-DDTHH:MM:SS): %w", err)
	}

	if publishAt.Before(time.Now()) {
		return fmt.Errorf("scheduled time must be in the future")
	}

	post.Status = storage.StatusScheduled
	post.PublishedAt = &publishAt

	if err := repo.Update(post); err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	fmt.Printf("Scheduled: %s for %s\n", post.Title, publishAt.Format("2006-01-02 15:04:05"))
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

		// Index post content for full-text search
		tagsStr := strings.Join(post.Tags, " ")
		if err := repo.IndexPost(dbPost.ID, post.Title, tagsStr, post.Content); err != nil {
			log.Printf("Warning: failed to index post %s: %v", post.Slug, err)
		}

		count++
	}

	return count, nil
}
