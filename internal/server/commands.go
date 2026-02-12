package server

import (
	"fmt"
	"io"
	"strings"

	"github.com/ajmeese7/termblog/internal/blog"
	"github.com/ajmeese7/termblog/internal/storage"
)

// CommandHandler handles non-interactive SSH commands
type CommandHandler struct {
	repo   *storage.PostRepository
	loader *blog.ContentLoader
	feed   *blog.FeedGenerator
}

// NewCommandHandler creates a new command handler
func NewCommandHandler(repo *storage.PostRepository, loader *blog.ContentLoader, feed *blog.FeedGenerator) *CommandHandler {
	return &CommandHandler{
		repo:   repo,
		loader: loader,
		feed:   feed,
	}
}

// HandleCommand processes a non-interactive SSH command
// Returns true if the command was handled, false if it should fall through to TUI
func (h *CommandHandler) HandleCommand(w io.Writer, args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil // No command, fall through to TUI
	}

	cmd := strings.ToLower(args[0])
	cmdArgs := args[1:]

	switch cmd {
	case "posts", "list", "ls":
		return true, h.handlePosts(w)
	case "read", "cat", "get":
		return true, h.handleRead(w, cmdArgs)
	case "rss", "feed":
		return true, h.handleRSS(w)
	case "search", "find":
		return true, h.handleSearch(w, cmdArgs)
	case "help", "-h", "--help":
		return true, h.handleHelp(w)
	default:
		return false, nil // Unknown command, fall through to TUI
	}
}

// handlePosts outputs a plain-text list of published posts
// Format: YYYY-MM-DD  Title  [tags]
func (h *CommandHandler) handlePosts(w io.Writer) error {
	posts, err := h.repo.ListPublished(1000, 0) // Get all published posts
	if err != nil {
		return fmt.Errorf("failed to list posts: %w", err)
	}

	if len(posts) == 0 {
		fmt.Fprintln(w, "No published posts found.")
		return nil
	}

	for _, post := range posts {
		// Get the date
		date := post.CreatedAt.Format("2006-01-02")
		if post.PublishedAt != nil {
			date = post.PublishedAt.Format("2006-01-02")
		}

		// Format tags
		tags := ""
		if len(post.Tags) > 0 {
			tags = "  [" + strings.Join(post.Tags, ", ") + "]"
		}

		// Output: YYYY-MM-DD  slug  title  [tags]
		fmt.Fprintf(w, "%s  %-20s  %s%s\n", date, post.Slug, post.Title, tags)
	}

	return nil
}

// handleRead outputs the raw markdown or rendered text of a post
// Usage: read <slug> [--rendered]
func (h *CommandHandler) handleRead(w io.Writer, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: read <slug> [--rendered]")
	}

	slug := args[0]
	rendered := false

	// Check for --rendered flag
	for _, arg := range args[1:] {
		if arg == "--rendered" || arg == "-r" {
			rendered = true
		}
	}

	// First check if post exists and is published in the database
	dbPost, err := h.repo.GetBySlug(slug)
	if err != nil {
		return fmt.Errorf("failed to get post: %w", err)
	}
	if dbPost == nil {
		return fmt.Errorf("post not found: %s", slug)
	}
	if dbPost.Status != storage.StatusPublished {
		return fmt.Errorf("post not found: %s", slug) // Don't reveal unpublished posts
	}

	// Load the full post content from filesystem
	post, err := h.loader.LoadBySlug(slug)
	if err != nil {
		return fmt.Errorf("failed to load post: %w", err)
	}
	if post == nil {
		return fmt.Errorf("post not found: %s", slug)
	}

	if rendered {
		// Render markdown to plain text (strip formatting)
		output := renderPlainText(post.Content)
		fmt.Fprintln(w, output)
	} else {
		// Output raw markdown
		fmt.Fprintln(w, post.Content)
	}

	return nil
}

// handleRSS outputs the RSS feed XML
func (h *CommandHandler) handleRSS(w io.Writer) error {
	if h.feed == nil {
		return fmt.Errorf("RSS feed not configured")
	}

	// Load published posts
	dbPosts, err := h.repo.ListPublished(50, 0)
	if err != nil {
		return fmt.Errorf("failed to list posts: %w", err)
	}

	// Convert to blog posts with content
	var posts []*blog.Post
	for _, dbPost := range dbPosts {
		post, err := h.loader.LoadBySlug(dbPost.Slug)
		if err != nil {
			continue // Skip posts that can't be loaded
		}
		if post != nil {
			posts = append(posts, post)
		}
	}

	rss, err := h.feed.GenerateRSS(posts)
	if err != nil {
		return fmt.Errorf("failed to generate RSS: %w", err)
	}

	fmt.Fprintln(w, rss)
	return nil
}

// handleSearch searches posts and outputs matching titles/slugs
// Usage: search <query>
func (h *CommandHandler) handleSearch(w io.Writer, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: search <query>")
	}

	query := strings.Join(args, " ")
	posts, err := h.repo.Search(query, 50)
	if err != nil {
		return fmt.Errorf("failed to search: %w", err)
	}

	if len(posts) == 0 {
		fmt.Fprintf(w, "No posts found matching: %s\n", query)
		return nil
	}

	fmt.Fprintf(w, "Found %d post(s) matching: %s\n\n", len(posts), query)

	for _, post := range posts {
		date := post.CreatedAt.Format("2006-01-02")
		if post.PublishedAt != nil {
			date = post.PublishedAt.Format("2006-01-02")
		}

		tags := ""
		if len(post.Tags) > 0 {
			tags = "  [" + strings.Join(post.Tags, ", ") + "]"
		}

		fmt.Fprintf(w, "%s  %-20s  %s%s\n", date, post.Slug, post.Title, tags)
	}

	return nil
}

// handleHelp outputs available commands
func (h *CommandHandler) handleHelp(w io.Writer) error {
	help := `TermBlog SSH Commands

Usage: ssh -p 2222 <host> <command> [arguments]

Commands:
  posts, list, ls         List all published posts
  read <slug>             Output raw markdown of a post
  read <slug> --rendered  Output plain text (stripped markdown)
  rss, feed               Output RSS feed XML
  search <query>          Search posts by title or tags
  help                    Show this help message

Examples:
  ssh -p 2222 blog.example.com posts
  ssh -p 2222 blog.example.com read my-first-post
  ssh -p 2222 blog.example.com read my-first-post --rendered | less
  ssh -p 2222 blog.example.com rss > feed.xml
  ssh -p 2222 blog.example.com search golang

Without a command, the interactive TUI is launched.
`
	fmt.Fprintln(w, help)
	return nil
}

// renderPlainText converts markdown to plain text by stripping formatting
func renderPlainText(markdown string) string {
	lines := strings.Split(markdown, "\n")
	var result []string

	for _, line := range lines {
		// Remove common markdown formatting
		line = strings.TrimSpace(line)

		// Remove heading markers
		if strings.HasPrefix(line, "#") {
			line = strings.TrimLeft(line, "# ")
		}

		// Remove bold/italic markers
		line = strings.ReplaceAll(line, "**", "")
		line = strings.ReplaceAll(line, "__", "")
		line = strings.ReplaceAll(line, "*", "")
		line = strings.ReplaceAll(line, "_", "")

		// Remove inline code markers
		line = strings.ReplaceAll(line, "`", "")

		// Remove link formatting [text](url) -> text
		for {
			start := strings.Index(line, "[")
			if start == -1 {
				break
			}
			end := strings.Index(line[start:], "](")
			if end == -1 {
				break
			}
			urlEnd := strings.Index(line[start+end:], ")")
			if urlEnd == -1 {
				break
			}
			// Extract text part
			text := line[start+1 : start+end]
			line = line[:start] + text + line[start+end+urlEnd+1:]
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}
