package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ajmeese7/termblog/internal/storage"
	"github.com/ajmeese7/termblog/internal/theme"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AdminModel handles the admin interface for managing posts
type AdminModel struct {
	repo       *storage.PostRepository
	viewRepo   *storage.ViewRepository
	styles     *theme.Styles
	keyMap     KeyMap
	contentDir string
	author     string

	posts     []*storage.Post
	viewStats map[int64]*storage.ViewStats
	cursor    int

	// Confirmation state (-1 means no confirmation active)
	confirmDeleteIdx int

	width  int
	height int

	err error
}

// NewAdminModel creates a new admin model
func NewAdminModel(repo *storage.PostRepository, viewRepo *storage.ViewRepository, styles *theme.Styles, contentDir, author string) *AdminModel {
	return &AdminModel{
		repo:             repo,
		viewRepo:         viewRepo,
		styles:           styles,
		keyMap:           DefaultKeyMap(),
		contentDir:       contentDir,
		author:           author,
		confirmDeleteIdx: -1,
	}
}

// SetSize sets the available dimensions
func (m *AdminModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetStyles updates the styles
func (m *AdminModel) SetStyles(styles *theme.Styles) {
	m.styles = styles
}

// Init loads the posts
func (m *AdminModel) Init() tea.Cmd {
	return m.loadPosts
}

// loadPosts fetches all posts from the repository
func (m *AdminModel) loadPosts() tea.Msg {
	posts, err := m.repo.ListAll(1000, 0)
	if err != nil {
		return adminErrorMsg{err: err}
	}

	// Load view stats if available
	var viewStats map[int64]*storage.ViewStats
	if m.viewRepo != nil {
		viewStats, _ = m.viewRepo.GetAllViewStats()
	}

	return adminPostsLoadedMsg{posts: posts, viewStats: viewStats}
}

// Update handles input and messages
func (m *AdminModel) Update(msg tea.Msg) (*AdminModel, tea.Cmd) {
	switch msg := msg.(type) {
	case adminPostsLoadedMsg:
		m.posts = msg.posts
		m.viewStats = msg.viewStats
		m.err = nil
		if m.cursor >= len(m.posts) {
			m.cursor = len(m.posts) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		return m, nil

	case adminErrorMsg:
		m.err = msg.err
		return m, nil

	case adminPostDeletedMsg:
		m.confirmDeleteIdx = -1
		return m, m.loadPosts

	case adminPostUpdatedMsg:
		return m, m.loadPosts

	case tea.KeyMsg:
		// Handle delete confirmation
		if m.confirmDeleteIdx >= 0 {
			switch msg.String() {
			case "y", "Y":
				if m.confirmDeleteIdx < len(m.posts) {
					return m, m.deletePost(m.posts[m.confirmDeleteIdx])
				}
			case "n", "N", "esc":
				m.confirmDeleteIdx = -1
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keyMap.Back):
			return m, func() tea.Msg { return AdminCloseMsg{} }

		case key.Matches(msg, m.keyMap.Up):
			if m.cursor > 0 {
				m.cursor--
			}

		case key.Matches(msg, m.keyMap.Down):
			if m.cursor < len(m.posts)-1 {
				m.cursor++
			}

		case msg.String() == "n":
			// New post
			return m, func() tea.Msg { return AdminNewPostMsg{} }

		case msg.String() == "e", key.Matches(msg, m.keyMap.Enter):
			// Edit post
			if len(m.posts) > 0 && m.cursor < len(m.posts) {
				return m, func() tea.Msg { return AdminEditPostMsg{Post: m.posts[m.cursor]} }
			}

		case msg.String() == "d":
			// Delete post (with confirmation)
			if len(m.posts) > 0 && m.cursor < len(m.posts) {
				m.confirmDeleteIdx = m.cursor
			}

		case msg.String() == "p":
			// Publish/Unpublish toggle
			if len(m.posts) > 0 && m.cursor < len(m.posts) {
				post := m.posts[m.cursor]
				return m, m.togglePublish(post)
			}

		case msg.String() == "g":
			// Go to top
			m.cursor = 0

		case msg.String() == "G":
			// Go to bottom
			if len(m.posts) > 0 {
				m.cursor = len(m.posts) - 1
			}
		}
	}

	return m, nil
}

// View renders the admin interface
func (m *AdminModel) View() string {
	var sections []string

	emptyLine := m.styles.ContentBg.Width(m.width).Render("")

	// Subtitle (app header already provides the main title)
	subtitle := m.styles.HelpDesc.Render("Post Management")
	sections = append(sections, subtitle, "")

	// Error message
	if m.err != nil {
		sections = append(sections, m.styles.StatusError.Render(fmt.Sprintf("Error: %v", m.err)))
		sections = append(sections, emptyLine)
	}

	// Posts list
	if len(m.posts) == 0 {
		sections = append(sections, m.styles.HelpDesc.Render("No posts found. Press 'n' to create one."))
	} else {
		// Calculate visible posts based on height
		visibleHeight := m.height - 10 // Account for header, footer, etc.
		if visibleHeight < 5 {
			visibleHeight = 5
		}
		linesPerPost := 2
		maxVisible := visibleHeight / linesPerPost

		// Calculate scroll window
		start := 0
		if m.cursor >= maxVisible {
			start = m.cursor - maxVisible + 1
		}
		end := start + maxVisible
		if end > len(m.posts) {
			end = len(m.posts)
		}

		for i := start; i < end; i++ {
			post := m.posts[i]
			sections = append(sections, m.renderPostItem(i, post))
		}

		// Scroll indicator
		if len(m.posts) > maxVisible {
			indicator := fmt.Sprintf("(%d/%d)", m.cursor+1, len(m.posts))
			sections = append(sections, m.styles.HelpDesc.Render(indicator))
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Pad each line with background color to fill full width,
	// replacing reset codes that would clear the themed background
	lines := strings.Split(content, "\n")
	bgCode := extractBgCode(m.styles.ContentBg)
	for i, line := range lines {
		line = "  " + line
		if bgCode != "" {
			line = strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+bgCode)
		}
		lines[i] = m.styles.ContentBg.Width(m.width).Render(line)
	}
	return strings.Join(lines, "\n")
}

// renderPostItem renders a single post in the list
func (m *AdminModel) renderPostItem(idx int, post *storage.Post) string {
	// Inline delete confirmation
	if idx == m.confirmDeleteIdx {
		confirmMsg := fmt.Sprintf("Delete '%s'? (y/n)", post.Title)
		line1 := m.styles.StatusError.Render("► " + confirmMsg)
		line2 := m.styles.ListDate.Render("    ")
		return lipgloss.JoinVertical(lipgloss.Left, line1, line2)
	}

	isSelected := idx == m.cursor

	// Status indicator
	var status string
	switch post.Status {
	case storage.StatusDraft:
		status = m.styles.HelpDesc.Render("[draft]")
	case storage.StatusPublished:
		status = m.styles.HelpKey.Render("[published]")
	case storage.StatusScheduled:
		if post.PublishedAt != nil {
			status = m.styles.StatusMessage.Render(fmt.Sprintf("[scheduled: %s]", post.PublishedAt.Format("2006-01-02")))
		} else {
			status = m.styles.StatusMessage.Render("[scheduled]")
		}
	}

	// Build title line with status inline
	// Use inline styling to avoid padding that forces line wraps
	var line1 string
	if isSelected {
		titleStyle := lipgloss.NewStyle().
			Foreground(m.styles.ListSelected.GetForeground()).
			Background(m.styles.ListSelected.GetBackground()).
			Bold(true)
		line1 = titleStyle.Render("► "+post.Title) + " " + status
	} else {
		titleStyle := lipgloss.NewStyle().
			Foreground(m.styles.ListItem.GetForeground()).
			Background(m.styles.ListItem.GetBackground())
		line1 = titleStyle.Render("  "+post.Title) + " " + status
	}

	// Date
	var dateStr string
	if post.PublishedAt != nil {
		dateStr = post.PublishedAt.Format("2006-01-02")
	} else {
		dateStr = post.CreatedAt.Format("2006-01-02")
	}

	// View stats
	var viewInfo string
	if m.viewStats != nil {
		if stats, ok := m.viewStats[post.ID]; ok {
			viewInfo = fmt.Sprintf(" | %d views (%d unique)", stats.TotalViews, stats.UniqueViewers)
		}
	}

	line2 := m.styles.ListDate.Render(fmt.Sprintf("    %s | %s%s", dateStr, post.Slug, viewInfo))

	return lipgloss.JoinVertical(lipgloss.Left, line1, line2)
}

// togglePublish changes the publish status of a post
func (m *AdminModel) togglePublish(post *storage.Post) tea.Cmd {
	return func() tea.Msg {
		if post.Status == storage.StatusPublished {
			post.Status = storage.StatusDraft
			post.PublishedAt = nil
		} else {
			post.Status = storage.StatusPublished
			now := time.Now()
			post.PublishedAt = &now
		}

		if err := m.repo.Update(post); err != nil {
			return adminErrorMsg{err: err}
		}
		return adminPostUpdatedMsg{}
	}
}

// deletePost removes a post from the database
func (m *AdminModel) deletePost(post *storage.Post) tea.Cmd {
	return func() tea.Msg {
		if err := m.repo.Delete(post.ID); err != nil {
			return adminErrorMsg{err: err}
		}
		return adminPostDeletedMsg{}
	}
}

// Messages

type adminPostsLoadedMsg struct {
	posts     []*storage.Post
	viewStats map[int64]*storage.ViewStats
}

type adminErrorMsg struct {
	err error
}

type adminPostDeletedMsg struct{}

type adminPostUpdatedMsg struct{}

// AdminCloseMsg is sent when admin wants to close the admin view
type AdminCloseMsg struct{}

// AdminNewPostMsg is sent when admin wants to create a new post
type AdminNewPostMsg struct{}

// AdminEditPostMsg is sent when admin wants to edit a post
type AdminEditPostMsg struct {
	Post *storage.Post
}
