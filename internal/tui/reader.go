package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/ajmeese7/termblog/internal/blog"
	"github.com/ajmeese7/termblog/internal/storage"
	"github.com/ajmeese7/termblog/internal/theme"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// ReaderModel handles the post reader view
type ReaderModel struct {
	styles   *theme.Styles
	keyMap   KeyMap
	viewport viewport.Model

	post     *storage.Post
	content  string
	rendered string

	width  int
	height int
	ready  bool
}

// NewReaderModel creates a new reader model
func NewReaderModel(styles *theme.Styles) *ReaderModel {
	return &ReaderModel{
		styles: styles,
		keyMap: DefaultKeyMap(),
	}
}

// SetSize updates the dimensions
func (m *ReaderModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	headerHeight := 4 // Title + meta + padding
	footerHeight := 1

	if !m.ready {
		m.viewport = viewport.New(width-4, height-headerHeight-footerHeight)
		m.viewport.Style = m.styles.Reader
		m.ready = true
	} else {
		m.viewport.Width = width - 4
		m.viewport.Height = height - headerHeight - footerHeight
	}

	// Re-render content if we have it
	if m.content != "" {
		m.renderContent()
	}
}

// SetPost sets the current post to display
func (m *ReaderModel) SetPost(post *storage.Post, content string) {
	m.post = post
	m.content = content
	m.viewport.GotoTop()
	m.renderContent()
}

// Update implements tea.Model
func (m *ReaderModel) Update(msg tea.Msg) (*ReaderModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Back):
			return m, func() tea.Msg { return BackToListMsg{} }
		case key.Matches(msg, m.keyMap.Search):
			return m, func() tea.Msg { return SearchActivatedMsg{} }
		case key.Matches(msg, m.keyMap.Up):
			m.viewport.LineUp(1)
		case key.Matches(msg, m.keyMap.Down):
			m.viewport.LineDown(1)
		case key.Matches(msg, m.keyMap.PageUp):
			m.viewport.ViewUp()
		case key.Matches(msg, m.keyMap.PageDown):
			m.viewport.ViewDown()
		case key.Matches(msg, m.keyMap.HalfUp):
			m.viewport.HalfViewUp()
		case key.Matches(msg, m.keyMap.HalfDown):
			m.viewport.HalfViewDown()
		case key.Matches(msg, m.keyMap.Top):
			m.viewport.GotoTop()
		case key.Matches(msg, m.keyMap.Bottom):
			m.viewport.GotoBottom()
		default:
			// Pass to viewport for scrolling
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m *ReaderModel) View() string {
	if m.post == nil {
		return m.styles.Reader.Render("No post selected")
	}

	// Build header
	title := m.styles.ReaderTitle.Render(m.post.Title)

	// Meta info
	var metaParts []string
	if m.post.PublishedAt != nil {
		metaParts = append(metaParts, m.post.PublishedAt.Format("January 2, 2006"))
	}
	if len(m.post.Tags) > 0 {
		metaParts = append(metaParts, strings.Join(m.post.Tags, ", "))
	}
	meta := m.styles.ReaderMeta.Render(strings.Join(metaParts, " • "))

	// Scroll indicator
	scrollPercent := m.viewport.ScrollPercent() * 100
	scrollInfo := m.styles.ReaderScroll.Render(
		fmt.Sprintf(" %.0f%% ", scrollPercent),
	)

	header := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		meta,
		"",
	)

	// Combine
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.Reader.Render(header),
		m.viewport.View(),
		scrollInfo,
	)
}

func (m *ReaderModel) renderContent() {
	if m.content == "" {
		m.rendered = ""
		m.viewport.SetContent("")
		return
	}

	// Create a glamour renderer with appropriate width
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(m.viewport.Width-2),
	)
	if err != nil {
		m.rendered = m.content
		m.viewport.SetContent(m.content)
		return
	}

	// Extract just the body (skip frontmatter)
	body := m.extractBody(m.content)

	rendered, err := renderer.Render(body)
	if err != nil {
		m.rendered = body
		m.viewport.SetContent(body)
		return
	}

	m.rendered = rendered
	m.viewport.SetContent(rendered)
}

func (m *ReaderModel) extractBody(content string) string {
	// Parse frontmatter and return just the body
	loader := blog.NewContentLoader("")
	post, err := loader.ParsePost(content, "")
	if err != nil {
		return content
	}
	return post.Content
}

// Helper function to load post content from file
func loadPostContent(filepath string) (string, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(content), nil
}
