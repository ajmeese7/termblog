package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/ajmeese7/termblog/internal/storage"
	"github.com/ajmeese7/termblog/internal/theme"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	postsPerPage = 10
)

// ListModel handles the post list view
type ListModel struct {
	repo   *storage.PostRepository
	styles *theme.Styles
	keyMap KeyMap
	title  string

	posts    []*storage.Post
	cursor   int
	offset   int
	total    int
	pageSize int

	width  int
	height int

	loading bool
	err     error
}

// NewListModel creates a new list model
func NewListModel(repo *storage.PostRepository, styles *theme.Styles, title string) *ListModel {
	return &ListModel{
		repo:     repo,
		styles:   styles,
		keyMap:   DefaultKeyMap(),
		title:    title,
		pageSize: postsPerPage,
	}
}

// Init implements tea.Model
func (m *ListModel) Init() tea.Cmd {
	return m.loadPosts()
}

// SetSize updates the dimensions
func (m *ListModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	// Adjust page size based on height (rough estimate: 3 lines per post)
	m.pageSize = max((height-4)/3, 5)
}

// Update implements tea.Model
func (m *ListModel) Update(msg tea.Msg) (*ListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case postsLoadedMsg:
		m.loading = false
		m.posts = msg.posts
		m.total = msg.total
		m.err = msg.err
		return m, nil

	case tea.MouseMsg:
		if m.loading {
			return m, nil
		}

		// Only handle left-click and scroll wheel, ignore everything else
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.moveCursor(-3)
			return m, nil
		case tea.MouseButtonWheelDown:
			m.moveCursor(3)
			return m, nil
		case tea.MouseButtonLeft:
			// Only handle left-click release (not press or motion)
			if msg.Action == tea.MouseActionRelease {
				// Calculate which post was clicked based on Y position
				// Header takes ~2 lines, each post takes 3 lines
				headerHeight := 2
				postHeight := 3
				clickedIdx := m.offset + (msg.Y-headerHeight)/postHeight

				if clickedIdx >= 0 && clickedIdx < len(m.posts) {
					if clickedIdx == m.cursor {
						// Click on already selected - enter post
						return m, m.selectPost()
					}
					m.cursor = clickedIdx
					m.adjustOffset()
				}
			}
			return m, nil
		default:
			// Ignore all other mouse events (right-click, middle-click, motion, etc.)
			return m, nil
		}

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keyMap.Up):
			m.moveCursor(-1)
		case key.Matches(msg, m.keyMap.Down):
			m.moveCursor(1)
		case key.Matches(msg, m.keyMap.PageUp):
			m.moveCursor(-m.pageSize)
		case key.Matches(msg, m.keyMap.PageDown):
			m.moveCursor(m.pageSize)
		case key.Matches(msg, m.keyMap.HalfUp):
			m.moveCursor(-m.pageSize / 2)
		case key.Matches(msg, m.keyMap.HalfDown):
			m.moveCursor(m.pageSize / 2)
		case key.Matches(msg, m.keyMap.Top):
			m.cursor = 0
			m.offset = 0
		case key.Matches(msg, m.keyMap.Bottom):
			m.cursor = len(m.posts) - 1
			m.adjustOffset()
		case key.Matches(msg, m.keyMap.Enter):
			return m, m.selectPost()
		case key.Matches(msg, m.keyMap.Search):
			return m, func() tea.Msg { return SearchActivatedMsg{} }
		}
	}

	return m, nil
}

// View implements tea.Model
func (m *ListModel) View() string {
	if m.loading {
		return m.styles.List.Render("Loading posts...")
	}

	if m.err != nil {
		return m.styles.List.Render(
			m.styles.StatusError.Render(fmt.Sprintf("Error: %v", m.err)),
		)
	}

	if len(m.posts) == 0 {
		return m.styles.List.Render(
			m.styles.HelpDesc.Render("No posts yet. Create one with: termblog new \"My First Post\""),
		)
	}

	var lines []string

	// Calculate visible range
	visibleEnd := min(m.offset+m.pageSize, len(m.posts))

	for i := m.offset; i < visibleEnd; i++ {
		post := m.posts[i]
		lines = append(lines, m.renderPost(i, post))
	}

	// Add scroll indicator
	if len(m.posts) > m.pageSize {
		scrollInfo := fmt.Sprintf(" %d-%d of %d ", m.offset+1, visibleEnd, len(m.posts))
		lines = append(lines, "")
		lines = append(lines, m.styles.ReaderScroll.Render(scrollInfo))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return m.styles.List.Width(m.width).Render(content)
}

func (m *ListModel) renderPost(idx int, post *storage.Post) string {
	isSelected := idx == m.cursor

	// Format date
	var dateStr string
	if post.PublishedAt != nil {
		dateStr = post.PublishedAt.Format("2006-01-02")
	} else {
		dateStr = post.CreatedAt.Format("2006-01-02")
	}

	// Format title with cursor indicator
	var title string
	if isSelected {
		title = m.styles.ListSelected.Render("► " + post.Title)
	} else {
		title = m.styles.ListItem.Render("  " + post.Title)
	}

	// Format date
	date := m.styles.ListDate.Render("  " + dateStr)

	// Format tags
	var tagsStr string
	if len(post.Tags) > 0 {
		tagsStr = m.styles.ListTags.Render("  [" + strings.Join(post.Tags, ", ") + "]")
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		date+tagsStr,
		"",
	)
}

func (m *ListModel) moveCursor(delta int) {
	m.cursor += delta

	// Clamp cursor
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.posts) {
		m.cursor = len(m.posts) - 1
	}

	m.adjustOffset()
}

func (m *ListModel) adjustOffset() {
	// Keep cursor visible within the viewport
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.pageSize {
		m.offset = m.cursor - m.pageSize + 1
	}

	// Clamp offset
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m *ListModel) loadPosts() tea.Cmd {
	m.loading = true
	return func() tea.Msg {
		posts, err := m.repo.ListPublished(100, 0)
		if err != nil {
			return postsLoadedMsg{err: err}
		}

		// Filter out posts whose files no longer exist
		validPosts := make([]*storage.Post, 0, len(posts))
		for _, post := range posts {
			if _, err := os.Stat(post.Filepath); err == nil {
				validPosts = append(validPosts, post)
			}
		}

		return postsLoadedMsg{
			posts: validPosts,
			total: len(validPosts),
		}
	}
}

func (m *ListModel) selectPost() tea.Cmd {
	if len(m.posts) == 0 || m.cursor >= len(m.posts) {
		return nil
	}

	post := m.posts[m.cursor]

	return func() tea.Msg {
		// Load the content from file
		content, err := loadPostContent(post.Filepath)
		if err != nil {
			return StatusMsg{
				Message: fmt.Sprintf("Failed to load post: %v", err),
				IsError: true,
			}
		}

		return PostSelectedMsg{
			Post:    post,
			Content: content,
		}
	}
}

// Reload refreshes the post list
func (m *ListModel) Reload() tea.Cmd {
	return m.loadPosts()
}

// Messages

type postsLoadedMsg struct {
	posts []*storage.Post
	total int
	err   error
}
