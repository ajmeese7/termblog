package tui

import (
	"github.com/ajmeese7/termblog/internal/blog"
	"github.com/ajmeese7/termblog/internal/storage"
	"github.com/ajmeese7/termblog/internal/theme"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// View represents the current view state
type View int

const (
	ViewList View = iota
	ViewReader
	ViewSearch
	ViewHelp
)

// KeyMap defines the key bindings
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	HalfUp   key.Binding
	HalfDown key.Binding
	Top      key.Binding
	Bottom   key.Binding
	Enter    key.Binding
	Back     key.Binding
	Search   key.Binding
	Help     key.Binding
	Quit     key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+b"),
			key.WithHelp("pgup/ctrl+b", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+f"),
			key.WithHelp("pgdn/ctrl+f", "page down"),
		),
		HalfUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "half page up"),
		),
		HalfDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "half page down"),
		),
		Top: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("g/home", "go to top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("G/end", "go to bottom"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter", "l"),
			key.WithHelp("enter/l", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "h", "backspace"),
			key.WithHelp("esc/h", "back"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// Config holds TUI configuration
type Config struct {
	BlogTitle string
	Author    string
}

// Model is the root Bubbletea model
type Model struct {
	// Dependencies
	repo   *storage.PostRepository
	loader *blog.ContentLoader
	styles *theme.Styles
	keyMap KeyMap
	config Config

	// View state
	currentView View
	prevView    View

	// Sub-models
	list   *ListModel
	reader *ReaderModel
	search *SearchModel

	// Window dimensions
	width  int
	height int

	// Status message
	statusMsg string
	isError   bool

	// Ready flag (after first resize)
	ready bool
}

// New creates a new root model
func New(repo *storage.PostRepository, loader *blog.ContentLoader, t *theme.Theme, cfg Config) *Model {
	styles := theme.NewStyles(t)

	m := &Model{
		repo:        repo,
		loader:      loader,
		styles:      styles,
		keyMap:      DefaultKeyMap(),
		config:      cfg,
		currentView: ViewList,
	}

	m.list = NewListModel(repo, styles, cfg.BlogTitle)
	m.reader = NewReaderModel(styles)
	m.search = NewSearchModel(repo, loader, styles)

	return m
}

// Init implements tea.Model
func (m *Model) Init() tea.Cmd {
	return m.list.Init()
}

// Update implements tea.Model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Propagate to sub-models
		m.list.SetSize(msg.Width, msg.Height-2) // Account for header/footer
		m.reader.SetSize(msg.Width, msg.Height-2)
		m.search.SetSize(msg.Width, msg.Height-2)

	case tea.KeyMsg:
		// Handle quit in any view
		if key.Matches(msg, m.keyMap.Quit) && m.currentView != ViewSearch {
			return m, tea.Quit
		}

		// Handle help toggle
		if key.Matches(msg, m.keyMap.Help) && m.currentView != ViewSearch {
			if m.currentView == ViewHelp {
				m.currentView = m.prevView
			} else {
				m.prevView = m.currentView
				m.currentView = ViewHelp
			}
			return m, nil
		}

	case PostSelectedMsg:
		// User selected a post in list view
		m.currentView = ViewReader
		m.reader.SetPost(msg.Post, msg.Content)
		return m, nil

	case BackToListMsg:
		// User pressed back in reader view
		m.currentView = ViewList
		return m, nil

	case SearchActivatedMsg:
		// User activated search
		m.prevView = m.currentView
		m.currentView = ViewSearch
		return m, m.search.Focus()

	case SearchCompletedMsg:
		// Search completed, show results or selected post
		if msg.SelectedPost != nil {
			m.currentView = ViewReader
			m.reader.SetPost(msg.SelectedPost, msg.Content)
		} else {
			m.currentView = m.prevView
		}
		return m, nil

	case SearchCancelledMsg:
		m.currentView = m.prevView
		return m, nil

	case StatusMsg:
		m.statusMsg = msg.Message
		m.isError = msg.IsError
		return m, nil
	}

	// Route updates to current view
	switch m.currentView {
	case ViewList:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)

	case ViewReader:
		var cmd tea.Cmd
		m.reader, cmd = m.reader.Update(msg)
		cmds = append(cmds, cmd)

	case ViewSearch:
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m *Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	var content string

	switch m.currentView {
	case ViewList:
		content = m.list.View()
	case ViewReader:
		content = m.reader.View()
	case ViewSearch:
		content = m.search.View()
	case ViewHelp:
		content = m.renderHelp()
	}

	// Build the full view with header and footer
	header := m.renderHeader()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		footer,
	)
}

func (m *Model) renderHeader() string {
	title := m.styles.Title.Render(m.config.BlogTitle)
	return m.styles.Header.Width(m.width).Render(title)
}

func (m *Model) renderFooter() string {
	var status string
	if m.statusMsg != "" {
		if m.isError {
			status = m.styles.StatusError.Render(m.statusMsg)
		} else {
			status = m.styles.StatusMessage.Render(m.statusMsg)
		}
	} else {
		status = m.renderHelpHint()
	}

	return m.styles.Footer.Width(m.width).Render(status)
}

func (m *Model) renderHelpHint() string {
	hints := []string{
		m.styles.HelpKey.Render("?") + m.styles.HelpDesc.Render(" help"),
		m.styles.HelpKey.Render("/") + m.styles.HelpDesc.Render(" search"),
		m.styles.HelpKey.Render("q") + m.styles.HelpDesc.Render(" quit"),
	}

	switch m.currentView {
	case ViewReader:
		hints = append([]string{
			m.styles.HelpKey.Render("esc") + m.styles.HelpDesc.Render(" back"),
		}, hints...)
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, hints...)
}

func (m *Model) renderHelp() string {
	sections := []string{
		m.styles.HelpSection.Render("Navigation"),
		m.renderHelpLine("j/↓", "Move down"),
		m.renderHelpLine("k/↑", "Move up"),
		m.renderHelpLine("ctrl+d", "Half page down"),
		m.renderHelpLine("ctrl+u", "Half page up"),
		m.renderHelpLine("ctrl+f/pgdn", "Page down"),
		m.renderHelpLine("ctrl+b/pgup", "Page up"),
		m.renderHelpLine("g/home", "Go to top"),
		m.renderHelpLine("G/end", "Go to bottom"),
		"",
		m.styles.HelpSection.Render("Actions"),
		m.renderHelpLine("enter/l", "Select/Open"),
		m.renderHelpLine("esc/h", "Back"),
		m.renderHelpLine("/", "Search"),
		m.renderHelpLine("?", "Toggle help"),
		m.renderHelpLine("q", "Quit"),
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return m.styles.Help.Render(content)
}

func (m *Model) renderHelpLine(key, desc string) string {
	k := m.styles.HelpKey.Render(key)
	d := m.styles.HelpDesc.Render(" - " + desc)
	return k + d
}

// Messages

// PostSelectedMsg is sent when a post is selected
type PostSelectedMsg struct {
	Post    *storage.Post
	Content string
}

// BackToListMsg is sent when user wants to go back to list
type BackToListMsg struct{}

// SearchActivatedMsg is sent when search is activated
type SearchActivatedMsg struct{}

// SearchCompletedMsg is sent when search is completed
type SearchCompletedMsg struct {
	SelectedPost *storage.Post
	Content      string
}

// SearchCancelledMsg is sent when search is cancelled
type SearchCancelledMsg struct{}

// StatusMsg is sent to update the status bar
type StatusMsg struct {
	Message string
	IsError bool
}
