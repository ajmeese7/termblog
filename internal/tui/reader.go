package tui

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/ajmeese7/termblog/internal/blog"
	"github.com/ajmeese7/termblog/internal/storage"
	"github.com/ajmeese7/termblog/internal/theme"
	"github.com/ajmeese7/termblog/internal/theme/styles"
	chromaStyles "github.com/alecthomas/chroma/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

// ReaderModel handles the post reader view
type ReaderModel struct {
	styles    *theme.Styles
	keyMap    KeyMap
	viewport  viewport.Model
	themeName string

	post    *storage.Post
	content string

	width  int
	height int
	ready  bool
}

// NewReaderModel creates a new reader model
func NewReaderModel(styles *theme.Styles, themeName string) *ReaderModel {
	return &ReaderModel{
		styles:    styles,
		keyMap:    DefaultKeyMap(),
		themeName: themeName,
	}
}

// SetTheme updates the theme and re-renders content
func (m *ReaderModel) SetTheme(styles *theme.Styles, themeName string) {
	m.styles = styles
	m.themeName = themeName
	// Re-render content with new theme
	if m.content != "" {
		m.renderContent()
	}
}

// SetSize updates the dimensions
func (m *ReaderModel) SetSize(width, height int) {
	m.width = width
	m.height = height

	headerHeight := 3 // Title + meta + blank line
	footerHeight := 1

	viewportHeight := height - headerHeight - footerHeight
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	if !m.ready {
		m.viewport = viewport.New(width, viewportHeight)
		m.viewport.KeyMap = viewport.KeyMap{} // Disable default keys, we handle them
		m.viewport.MouseWheelEnabled = true
		m.viewport.SetYOffset(0)
		m.ready = true
	} else {
		m.viewport.Width = width
		m.viewport.Height = viewportHeight
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
	var cmd tea.Cmd

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
		case key.Matches(msg, m.keyMap.HalfUp):
			m.viewport.HalfViewUp()
		case key.Matches(msg, m.keyMap.HalfDown):
			m.viewport.HalfViewDown()
		case key.Matches(msg, m.keyMap.PageUp):
			m.viewport.ViewUp()
		case key.Matches(msg, m.keyMap.PageDown):
			m.viewport.ViewDown()
		case key.Matches(msg, m.keyMap.Top):
			m.viewport.GotoTop()
		case key.Matches(msg, m.keyMap.Bottom):
			m.viewport.GotoBottom()
		}

	case tea.MouseMsg:
		// Let viewport handle mouse events
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View implements tea.Model
func (m *ReaderModel) View() string {
	if m.post == nil {
		return m.styles.Reader.Render("No post selected")
	}

	// Build header with full-width background
	title := m.styles.ReaderTitle.Width(m.width).Render(m.post.Title)

	var metaParts []string
	if m.post.PublishedAt != nil {
		metaParts = append(metaParts, m.post.PublishedAt.Format("January 2, 2006"))
	}
	if len(m.post.Tags) > 0 {
		metaParts = append(metaParts, strings.Join(m.post.Tags, ", "))
	}
	meta := m.styles.ReaderMeta.Width(m.width).Render(strings.Join(metaParts, " • "))

	// Empty line with background
	emptyLine := m.styles.ContentBg.Width(m.width).Render("")

	// Scroll indicator - fixed width to prevent redraw flicker
	scrollPercent := m.viewport.ScrollPercent() * 100
	scrollInfo := m.styles.ReaderScroll.Width(m.width).Render(fmt.Sprintf(" %3.0f%% ", scrollPercent))

	// Manually join with newlines instead of lipgloss.JoinVertical
	// This gives bubbletea's renderer consistent line counts
	return title + "\n" + meta + "\n" + emptyLine + "\n" + m.viewport.View() + "\n" + scrollInfo
}

func (m *ReaderModel) renderContent() {
	if m.content == "" {
		m.viewport.SetContent("")
		return
	}

	contentWidth := m.width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	// Clear cached chroma style so glamour re-registers with current theme's colors.
	// Glamour v0.6.0 caches under the fixed name "charm" and skips re-registration.
	delete(chromaStyles.Registry, "charm")

	// Load custom Glamour style for current theme, or fall back to auto-detection
	var renderer *glamour.TermRenderer
	styleJSON, err := styles.GetStyle(m.themeName)
	if err != nil {
		// No custom Glamour style — use auto detection (adapts to terminal)
		renderer, err = glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(contentWidth),
		)
	} else {
		renderer, err = glamour.NewTermRenderer(
			glamour.WithStylesFromJSONBytes(styleJSON),
			glamour.WithWordWrap(contentWidth),
		)
	}
	if err != nil {
		m.viewport.SetContent(m.content)
		return
	}

	// Extract just the body (skip frontmatter)
	body := m.extractBody(m.content)

	// Generate and prepend table of contents if there are headings
	if toc := generateTOC(body); toc != "" {
		body = toc + "\n" + body
	}

	// Preprocess to fix nested blockquotes (glamour doesn't support them)
	body = preprocessNestedBlockquotes(body)

	rendered, err := renderer.Render(body)
	if err != nil {
		m.viewport.SetContent(body)
		return
	}

	// Pad each line with background color to fill full terminal width.
	// Glamour word-wraps at contentWidth for readability, but the background
	// must extend to m.width to avoid uncolored gaps at the right edge.
	rendered = m.padContentLines(rendered, m.width)

	m.viewport.SetContent(rendered)
}

// padContentLines applies the theme background to each line of content
// It also fixes the issue where glamour's [0m reset codes clear the background
func (m *ReaderModel) padContentLines(content string, width int) string {
	// Get the background color from styles by rendering an empty string
	// and extracting the ANSI code
	bgSample := m.styles.ContentBg.Render("")
	// Extract background code (e.g., "\x1b[48;2;40;42;54m")
	bgCode := ""
	if idx := strings.Index(bgSample, "\x1b[48;"); idx >= 0 {
		endIdx := strings.Index(bgSample[idx:], "m")
		if endIdx > 0 {
			bgCode = bgSample[idx : idx+endIdx+1]
		}
	}

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		// Replace [0m (reset) with [0m + background code to preserve background
		if bgCode != "" {
			line = strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+bgCode)
		}
		lines[i] = m.styles.ContentBg.Width(width).Render(line)
	}
	return strings.Join(lines, "\n")
}

func (m *ReaderModel) extractBody(content string) string {
	loader := blog.NewContentLoader("")
	post, err := loader.ParsePost(content, "")
	if err != nil {
		return content
	}
	return post.Content
}

// preprocessNestedBlockquotes converts nested blockquotes to a format glamour can handle
func preprocessNestedBlockquotes(content string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		// Count the number of > at the start of the line
		trimmed := strings.TrimLeft(line, " \t")
		level := 0
		for strings.HasPrefix(trimmed, ">") {
			level++
			trimmed = strings.TrimPrefix(trimmed, ">")
			trimmed = strings.TrimLeft(trimmed, " ")
		}

		if level > 1 {
			// Convert nested quote to indented text with visual markers
			// Use │ characters for each nesting level
			prefix := strings.Repeat("│ ", level-1)
			result = append(result, "> "+prefix+trimmed)
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// tocHeading represents a heading extracted from markdown
type tocHeading struct {
	level int
	text  string
}

// headingRegex matches ATX-style markdown headings (## Heading)
var headingRegex = regexp.MustCompile(`^(#{2,6})\s+(.+)$`)
var orderedListMarkerRegex = regexp.MustCompile(`^(\d+)\.\s+(.+)$`)

// formatTOCHeadingText avoids nested ordered-list parsing for numbered headings.
// We replace the marker space with "&nbsp;" so markdown keeps it as plain text.
func formatTOCHeadingText(text string) string {
	matches := orderedListMarkerRegex.FindStringSubmatch(text)
	if matches == nil {
		return text
	}
	return fmt.Sprintf("%s.&nbsp;%s", matches[1], matches[2])
}

// generateTOC extracts headings from markdown and generates a table of contents
// Only generates TOC if there are 2 or more headings
func generateTOC(content string) string {
	lines := strings.Split(content, "\n")
	var headings []tocHeading
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track code blocks to avoid matching headings inside them
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		if matches := headingRegex.FindStringSubmatch(trimmed); matches != nil {
			level := len(matches[1]) // Number of # characters
			text := strings.TrimSpace(matches[2])
			headings = append(headings, tocHeading{level: level, text: text})
		}
	}

	// Only show TOC if there are enough headings to be useful
	if len(headings) < 2 {
		return ""
	}

	// Find the minimum heading level to normalize indentation
	minLevel := 6
	for _, h := range headings {
		if h.level < minLevel {
			minLevel = h.level
		}
	}

	// Build the TOC as a markdown list.
	// Numbered headings are normalized so they don't get parsed as nested ordered lists.
	var toc strings.Builder
	toc.WriteString("**Contents**\n\n")
	for _, h := range headings {
		indent := strings.Repeat("  ", h.level-minLevel)
		toc.WriteString(fmt.Sprintf("%s- %s\n", indent, formatTOCHeadingText(h.text)))
	}

	return toc.String()
}

// Helper function to load post content from file
func loadPostContent(filepath string) (string, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(content), nil
}
