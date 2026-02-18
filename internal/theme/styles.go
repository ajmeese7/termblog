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

	// Content background - used for padding content lines to full width
	ContentBg lipgloss.Style

	// Renderer used to create these styles (for inline style creation)
	Renderer *lipgloss.Renderer
}

// NewStyles creates styles from a theme.
// If r is nil, the default lipgloss renderer is used.
func NewStyles(theme *Theme, r *lipgloss.Renderer) *Styles {
	if r == nil {
		r = lipgloss.DefaultRenderer()
	}

	c := theme.Colors

	primary := lipgloss.Color(c.Primary)
	secondary := lipgloss.Color(c.Secondary)
	background := lipgloss.Color(c.Background)
	text := lipgloss.Color(c.Text)
	muted := lipgloss.Color(c.Muted)
	accent := lipgloss.Color(c.Accent)
	errorColor := lipgloss.Color(c.Error)
	success := lipgloss.Color(c.Success)
	border := lipgloss.Color(c.Border)

	return &Styles{
		// Base styles
		App: r.NewStyle().
			Foreground(text).
			Background(background),

		Header: r.NewStyle().
			Foreground(primary).
			Background(background).
			Bold(true).
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(border),

		Footer: r.NewStyle().
			Foreground(muted).
			Background(background).
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(border),

		// List view
		List: r.NewStyle().
			Background(background).
			Padding(1, 2),

		ListItem: r.NewStyle().
			Foreground(text).
			Background(background).
			Padding(0, 1),

		ListSelected: r.NewStyle().
			Foreground(primary).
			Bold(true).
			Background(lipgloss.Color(c.Border)).
			Padding(0, 1),

		ListTitle: r.NewStyle().
			Foreground(primary).
			Background(background).
			Bold(true),

		ListDate: r.NewStyle().
			Foreground(muted).
			Background(background).
			Italic(true),

		ListTags: r.NewStyle().
			Foreground(secondary).
			Background(background),

		// Reader view
		Reader: r.NewStyle().
			Background(background).
			Padding(1, 2),

		ReaderTitle: r.NewStyle().
			Foreground(primary).
			Background(background).
			Bold(true).
			PaddingLeft(2),

		ReaderMeta: r.NewStyle().
			Foreground(muted).
			Background(background).
			Italic(true).
			PaddingLeft(2),

		ReaderScroll: r.NewStyle().
			Foreground(muted).
			Background(background),

		// Search view
		Search: r.NewStyle().
			Background(background).
			Padding(1, 2),

		SearchInput: r.NewStyle().
			Foreground(primary).
			Background(background).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(border).
			Padding(0, 1),

		SearchHint: r.NewStyle().
			Foreground(muted).
			Background(background).
			Italic(true),

		// Help
		Help: r.NewStyle().
			Foreground(text).
			Background(background).
			Padding(1, 2),

		HelpKey: r.NewStyle().
			Foreground(accent).
			Background(background).
			Bold(true),

		HelpDesc: r.NewStyle().
			Foreground(muted).
			Background(background),

		HelpSection: r.NewStyle().
			Foreground(primary).
			Background(background).
			Bold(true).
			MarginTop(1).
			MarginBottom(1),

		// Status
		StatusBar: r.NewStyle().
			Foreground(text).
			Background(background).
			Padding(0, 1),

		StatusMessage: r.NewStyle().
			Foreground(text).
			Background(background),

		StatusError: r.NewStyle().
			Foreground(errorColor).
			Background(background).
			Bold(true),

		StatusSuccess: r.NewStyle().
			Foreground(success).
			Background(background).
			Bold(true),

		// Misc
		Border: r.NewStyle().
			Background(background).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(border),

		Title: r.NewStyle().
			Foreground(primary).
			Background(background).
			Bold(true),

		Spinner: r.NewStyle().
			Foreground(accent).
			Background(background),

		ContentBg: r.NewStyle().
			Background(background),

		Renderer: r,
	}
}

// GlamourStyle returns a Glamour style name appropriate for the theme
func GlamourStyle(theme *Theme) string {
	// Use auto to let Glamour detect dark/light mode
	// In the future, we could create custom glamour styles based on the theme
	return "auto"
}
