package tui

import (
	"fmt"
	"strings"

	"github.com/ajmeese7/termblog/internal/theme"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ThemeSelectorModel is a model for selecting themes with preview
type ThemeSelectorModel struct {
	themes      []*theme.Theme
	themeNames  []string
	cursor      int
	selected    int // Currently active theme
	styles      *theme.Styles
	width       int
	height      int
	keyMap      KeyMap
}

// NewThemeSelectorModel creates a new theme selector
func NewThemeSelectorModel(themes []*theme.Theme, themeNames []string, currentIndex int, styles *theme.Styles, keyMap KeyMap) *ThemeSelectorModel {
	return &ThemeSelectorModel{
		themes:     themes,
		themeNames: themeNames,
		cursor:     currentIndex,
		selected:   currentIndex,
		styles:     styles,
		keyMap:     keyMap,
	}
}

// SetSize sets the available dimensions
func (m *ThemeSelectorModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetStyles updates the styles
func (m *ThemeSelectorModel) SetStyles(styles *theme.Styles) {
	m.styles = styles
}

// CurrentTheme returns the currently highlighted theme
func (m *ThemeSelectorModel) CurrentTheme() *theme.Theme {
	return m.themes[m.cursor]
}

// CurrentThemeName returns the name of the currently highlighted theme
func (m *ThemeSelectorModel) CurrentThemeName() string {
	return m.themeNames[m.cursor]
}

// SelectedIndex returns the index of the selected theme
func (m *ThemeSelectorModel) SelectedIndex() int {
	return m.cursor
}

// Update handles input
func (m *ThemeSelectorModel) Update(msg tea.Msg) (*ThemeSelectorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Up):
			if m.cursor > 0 {
				m.cursor--
				return m, func() tea.Msg { return ThemePreviewMsg{Theme: m.themes[m.cursor], Name: m.themeNames[m.cursor]} }
			}
		case key.Matches(msg, m.keyMap.Down):
			if m.cursor < len(m.themes)-1 {
				m.cursor++
				return m, func() tea.Msg { return ThemePreviewMsg{Theme: m.themes[m.cursor], Name: m.themeNames[m.cursor]} }
			}
		case key.Matches(msg, m.keyMap.Enter):
			m.selected = m.cursor
			return m, func() tea.Msg { return ThemeSelectedMsg{Theme: m.themes[m.cursor], Name: m.themeNames[m.cursor]} }
		case key.Matches(msg, m.keyMap.Back):
			// Restore original selection on cancel
			return m, func() tea.Msg { return ThemeCancelledMsg{} }
		}
	}
	return m, nil
}

// View renders the theme selector
func (m *ThemeSelectorModel) View() string {
	var lines []string
	currentTheme := m.themes[m.cursor]
	bg := lipgloss.Color(currentTheme.Colors.Background)
	contentWidth := m.width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	// Background style for padding lines
	bgStyle := lipgloss.NewStyle().Background(bg)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(currentTheme.Colors.Accent)).
		Background(bg)

	lines = append(lines, titleStyle.Width(contentWidth).Render("Select Theme"))
	lines = append(lines, bgStyle.Width(contentWidth).Render(""))

	for i, t := range m.themes {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		selected := ""
		if i == m.selected {
			selected = " ✓"
		}

		// Create theme preview colors
		itemStyle := lipgloss.NewStyle().Background(bg)
		if i == m.cursor {
			itemStyle = itemStyle.
				Bold(true).
				Foreground(lipgloss.Color(t.Colors.Primary))
		} else {
			itemStyle = itemStyle.
				Foreground(lipgloss.Color(currentTheme.Colors.Muted))
		}

		descStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(currentTheme.Colors.Muted)).
			Background(bg).
			Italic(true)

		line := fmt.Sprintf("%s%s%s", cursor, t.Name, selected)
		lines = append(lines, itemStyle.Width(contentWidth).Render(line))

		if i == m.cursor {
			lines = append(lines, descStyle.Width(contentWidth).Render("    "+t.Description))
			lines = append(lines, bgStyle.Width(contentWidth).Render(m.renderColorPreview(t)))
		}
	}

	lines = append(lines, bgStyle.Width(contentWidth).Render(""))

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(currentTheme.Colors.Muted)).
		Background(bg)

	lines = append(lines, helpStyle.Width(contentWidth).Render("↑/↓ navigate • enter select • esc cancel"))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderColorPreview shows a visual preview of theme colors
func (m *ThemeSelectorModel) renderColorPreview(t *theme.Theme) string {
	colors := []struct {
		name  string
		color string
	}{
		{"Pri", t.Colors.Primary},
		{"Sec", t.Colors.Secondary},
		{"Acc", t.Colors.Accent},
		{"Txt", t.Colors.Text},
		{"Mut", t.Colors.Muted},
	}

	var blocks []string
	for _, c := range colors {
		style := lipgloss.NewStyle().
			Background(lipgloss.Color(c.color)).
			Foreground(lipgloss.Color(t.Colors.Background)).
			Padding(0, 1)
		blocks = append(blocks, style.Render(c.name))
	}

	return "    " + strings.Join(blocks, " ")
}

// ThemePreviewMsg is sent when cursor moves to preview a theme
type ThemePreviewMsg struct {
	Theme *theme.Theme
	Name  string
}

// ThemeSelectedMsg is sent when a theme is selected
type ThemeSelectedMsg struct {
	Theme *theme.Theme
	Name  string
}

// ThemeCancelledMsg is sent when theme selection is cancelled
type ThemeCancelledMsg struct{}
