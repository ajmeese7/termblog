package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// Styles holds all the Lipgloss styles for the TUI
type Styles struct {
	// Base styles
	App    lipgloss.Style
	Header lipgloss.Style
	Footer lipgloss.Style

	// List view
	List         lipgloss.Style
	ListItem     lipgloss.Style
	ListSelected lipgloss.Style
	ListTitle    lipgloss.Style
	ListDate     lipgloss.Style
	ListTags     lipgloss.Style

	// Reader view
	Reader       lipgloss.Style
	ReaderTitle  lipgloss.Style
	ReaderMeta   lipgloss.Style
	ReaderScroll lipgloss.Style

	// Search view
	Search      lipgloss.Style
	SearchInput lipgloss.Style
	SearchHint  lipgloss.Style

	// Help
	Help        lipgloss.Style
	HelpKey     lipgloss.Style
	HelpDesc    lipgloss.Style
	HelpSection lipgloss.Style

	// Status
	StatusBar     lipgloss.Style
	StatusMessage lipgloss.Style
	StatusError   lipgloss.Style
	StatusSuccess lipgloss.Style

	// Misc
	Border  lipgloss.Style
	Title   lipgloss.Style
	Spinner lipgloss.Style
}

// NewStyles creates styles from a theme
func NewStyles(theme *Theme) *Styles {
	c := theme.Colors

	primary := lipgloss.Color(c.Primary)
	secondary := lipgloss.Color(c.Secondary)
	text := lipgloss.Color(c.Text)
	muted := lipgloss.Color(c.Muted)
	accent := lipgloss.Color(c.Accent)
	errorColor := lipgloss.Color(c.Error)
	success := lipgloss.Color(c.Success)
	border := lipgloss.Color(c.Border)

	return &Styles{
		// Base styles
		App: lipgloss.NewStyle().
			Foreground(text),

		Header: lipgloss.NewStyle().
			Foreground(primary).
			Bold(true).
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(border),

		Footer: lipgloss.NewStyle().
			Foreground(muted).
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(border),

		// List view
		List: lipgloss.NewStyle().
			Padding(1, 2),

		ListItem: lipgloss.NewStyle().
			Foreground(text).
			Padding(0, 1),

		ListSelected: lipgloss.NewStyle().
			Foreground(primary).
			Bold(true).
			Background(lipgloss.Color(c.Border)).
			Padding(0, 1),

		ListTitle: lipgloss.NewStyle().
			Foreground(primary).
			Bold(true),

		ListDate: lipgloss.NewStyle().
			Foreground(muted).
			Italic(true),

		ListTags: lipgloss.NewStyle().
			Foreground(secondary),

		// Reader view
		Reader: lipgloss.NewStyle().
			Padding(1, 2),

		ReaderTitle: lipgloss.NewStyle().
			Foreground(primary).
			Bold(true).
			Padding(1, 0).
			MarginBottom(1),

		ReaderMeta: lipgloss.NewStyle().
			Foreground(muted).
			Italic(true).
			MarginBottom(1),

		ReaderScroll: lipgloss.NewStyle().
			Foreground(muted),

		// Search view
		Search: lipgloss.NewStyle().
			Padding(1, 2),

		SearchInput: lipgloss.NewStyle().
			Foreground(primary).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(border).
			Padding(0, 1),

		SearchHint: lipgloss.NewStyle().
			Foreground(muted).
			Italic(true),

		// Help
		Help: lipgloss.NewStyle().
			Foreground(text).
			Padding(1, 2),

		HelpKey: lipgloss.NewStyle().
			Foreground(accent).
			Bold(true),

		HelpDesc: lipgloss.NewStyle().
			Foreground(muted),

		HelpSection: lipgloss.NewStyle().
			Foreground(primary).
			Bold(true).
			MarginTop(1).
			MarginBottom(1),

		// Status
		StatusBar: lipgloss.NewStyle().
			Foreground(text).
			Padding(0, 1),

		StatusMessage: lipgloss.NewStyle().
			Foreground(text),

		StatusError: lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true),

		StatusSuccess: lipgloss.NewStyle().
			Foreground(success).
			Bold(true),

		// Misc
		Border: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(border),

		Title: lipgloss.NewStyle().
			Foreground(primary).
			Bold(true),

		Spinner: lipgloss.NewStyle().
			Foreground(accent),
	}
}

// GlamourStyle returns a Glamour style name appropriate for the theme
func GlamourStyle(theme *Theme) string {
	// Use auto to let Glamour detect dark/light mode
	// In the future, we could create custom glamour styles based on the theme
	return "auto"
}
