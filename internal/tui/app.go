package tui

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

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
	ViewThemeSelector
	ViewAdmin
	ViewEditor
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
	BlogTitle   string
	Author      string
	ASCIIHeader string // Optional ASCII art for header
	ContentDir  string // Path to content directory (for admin editing)
}

// Model is the root Bubbletea model
type Model struct {
	// Dependencies
	repo     *storage.PostRepository
	prefRepo *storage.PreferenceRepository
	viewRepo *storage.ViewRepository
	loader   *blog.ContentLoader
	styles   *theme.Styles
	keyMap   KeyMap
	config   Config
	renderer *lipgloss.Renderer // Session-specific renderer (nil = default)

	// Theme state
	themes      []*theme.Theme
	themeNames  []string
	themeIndex  int
	fingerprint string // SSH key fingerprint for theme persistence and view tracking

	// Admin state
	isAdmin bool // Whether the current user has admin privileges

	// View state
	currentView   View
	prevView      View
	savedPrevView View // stashed prevView when an overlay (admin/help/theme/search) opens
	searchOrigin  View // which view search was opened from (List or Reader)

	// Saved reader state — preserved when search is opened from reader,
	// restored when search ESC returns to reader
	savedReaderPost    *storage.Post
	savedReaderContent string

	// Sub-models
	list          *ListModel
	reader        *ReaderModel
	search        *SearchModel
	themeSelector *ThemeSelectorModel
	admin         *AdminModel
	editor        *EditorModel

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
	return NewWithPreferences(repo, loader, t, cfg, "", nil, nil, false, nil)
}

// NewWithPreferences creates a new root model with theme persistence and view tracking support.
// Pass a non-nil renderer for SSH sessions; nil uses the default renderer.
func NewWithPreferences(repo *storage.PostRepository, loader *blog.ContentLoader, t *theme.Theme, cfg Config, fingerprint string, prefRepo *storage.PreferenceRepository, viewRepo *storage.ViewRepository, isAdmin bool, r *lipgloss.Renderer) *Model {
	styles := theme.NewStyles(t, r)

	// Build theme list for cycling (includes all built-in themes)
	themeMap := theme.DefaultThemes()
	themeNames := []string{"pipboy", "dracula", "nord", "monokai", "monochrome", "amber", "matrix", "paper", "terminal"}
	themes := make([]*theme.Theme, len(themeNames))
	currentIndex := 0
	for i, name := range themeNames {
		themes[i] = themeMap[name]
		if themes[i].Name == t.Name {
			currentIndex = i
		}
	}

	m := &Model{
		repo:        repo,
		prefRepo:    prefRepo,
		viewRepo:    viewRepo,
		loader:      loader,
		styles:      styles,
		keyMap:      DefaultKeyMap(),
		config:      cfg,
		renderer:    r,
		themes:      themes,
		themeNames:  themeNames,
		themeIndex:  currentIndex,
		fingerprint: fingerprint,
		isAdmin:     isAdmin,
		currentView: ViewList,
	}

	m.list = NewListModel(repo, styles, cfg.BlogTitle)
	m.reader = NewReaderModel(styles, themeNames[currentIndex])
	m.search = NewSearchModel(repo, loader, styles)
	m.themeSelector = NewThemeSelectorModel(themes, themeNames, currentIndex, styles, m.keyMap)
	m.admin = NewAdminModel(repo, viewRepo, styles, cfg.ContentDir, cfg.Author)
	m.editor = NewEditorModel(styles)

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
	case postsLoadedMsg:
		// Always route to list model regardless of current view
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Propagate to sub-models
		m.list.SetSize(msg.Width, msg.Height-2) // Account for header/footer
		m.reader.SetSize(msg.Width, msg.Height-2)
		m.search.SetSize(msg.Width, msg.Height-2)
		m.themeSelector.SetSize(msg.Width, msg.Height-2)
		m.admin.SetSize(msg.Width, msg.Height-2)
		m.editor.SetSize(msg.Width, msg.Height-2)

	case tea.KeyMsg:
		// Editor view handles all its own keys - don't intercept
		if m.currentView == ViewEditor {
			var cmd tea.Cmd
			m.editor, cmd = m.editor.Update(msg)
			return m, cmd
		}

		// Handle quit in any view
		if key.Matches(msg, m.keyMap.Quit) && m.currentView != ViewSearch {
			return m, tea.Quit
		}

		// Handle theme toggle (t key) - open theme selector
		if msg.String() == "t" && m.currentView != ViewSearch && m.currentView != ViewThemeSelector {
			m.savedPrevView = m.prevView
			m.prevView = m.currentView
			m.currentView = ViewThemeSelector
			return m, tea.ClearScreen
		}

		// Handle help toggle
		if key.Matches(msg, m.keyMap.Help) && m.currentView != ViewSearch {
			if m.currentView == ViewHelp {
				m.currentView = m.prevView
				m.prevView = m.savedPrevView
			} else {
				m.savedPrevView = m.prevView
				m.prevView = m.currentView
				m.currentView = ViewHelp
			}
			return m, tea.ClearScreen
		}

		// Handle escape to close help
		if key.Matches(msg, m.keyMap.Back) && m.currentView == ViewHelp {
			m.currentView = m.prevView
			m.prevView = m.savedPrevView
			return m, tea.ClearScreen
		}

		// Handle admin toggle (a key) - only for admins, not in search/admin views
		if msg.String() == "a" && m.isAdmin && m.currentView != ViewSearch && m.currentView != ViewAdmin {
			m.savedPrevView = m.prevView
			m.prevView = m.currentView
			m.currentView = ViewAdmin
			return m, tea.Batch(tea.ClearScreen, m.admin.Init())
		}

	case PostSelectedMsg:
		// User selected a post in list view
		m.prevView = ViewList
		m.currentView = ViewReader
		m.reader.SetPost(msg.Post, msg.Content)
		// Record view for analytics
		if m.viewRepo != nil && msg.Post != nil {
			viewerHash := m.fingerprint
			if viewerHash == "" {
				viewerHash = "anonymous"
			}
			// Record view asynchronously to avoid blocking
			go func(postID int64, hash string) {
				if err := m.viewRepo.RecordView(postID, hash); err != nil {
					log.Printf("Failed to record view: %v", err)
				}
			}(msg.Post.ID, viewerHash)
		}
		return m, tea.ClearScreen

	case BackToListMsg:
		// User pressed back in reader view — return to wherever they came from
		m.currentView = m.prevView
		if m.currentView == ViewSearch {
			// Returning to search after viewing a search result.
			// Restore search's return chain: search ESC → searchOrigin → List
			m.prevView = m.searchOrigin
			m.savedPrevView = ViewList
		}
		return m, tea.ClearScreen

	case SearchActivatedMsg:
		// User activated search — save prevView so search is a proper overlay
		m.searchOrigin = m.currentView
		m.savedPrevView = m.prevView
		m.prevView = m.currentView
		// Save reader state so it can be restored when search closes
		if m.currentView == ViewReader && m.reader.post != nil {
			m.savedReaderPost = m.reader.post
			m.savedReaderContent = m.reader.content
		}
		m.currentView = ViewSearch
		return m, tea.Batch(tea.ClearScreen, m.search.Focus())

	case SearchCompletedMsg:
		// Search completed, show results or selected post
		if msg.SelectedPost != nil {
			// Reader's back goes to search; search's back goes to savedPrevView origin
			m.prevView = ViewSearch
			m.currentView = ViewReader
			m.reader.SetPost(msg.SelectedPost, msg.Content)
		} else {
			m.currentView = m.prevView
			m.prevView = m.savedPrevView
		}
		return m, tea.ClearScreen

	case SearchCancelledMsg:
		m.currentView = m.prevView
		m.prevView = m.savedPrevView
		// Restore saved reader post when returning to reader after search
		if m.currentView == ViewReader && m.savedReaderPost != nil {
			m.reader.SetPost(m.savedReaderPost, m.savedReaderContent)
			m.savedReaderPost = nil
			m.savedReaderContent = ""
		}
		return m, tea.ClearScreen

	case StatusMsg:
		m.statusMsg = msg.Message
		m.isError = msg.IsError
		return m, nil

	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil

	case ThemePreviewMsg:
		// Preview the theme while browsing
		m.styles = theme.NewStyles(msg.Theme, m.renderer)
		m.themeSelector.SetStyles(m.styles)
		m.admin.SetStyles(m.styles)
		m.editor.SetStyles(m.styles)
		return m, nil

	case ThemeSelectedMsg:
		// Apply the selected theme
		m.styles = theme.NewStyles(msg.Theme, m.renderer)
		m.themeIndex = m.themeSelector.SelectedIndex()
		m.list.styles = m.styles
		m.reader.SetTheme(m.styles, msg.Name)
		m.search.styles = m.styles
		m.themeSelector.SetStyles(m.styles)
		m.admin.SetStyles(m.styles)
		m.editor.SetStyles(m.styles)
		m.currentView = m.prevView
		m.prevView = m.savedPrevView

		// Save theme preference
		if m.fingerprint != "" && m.prefRepo != nil {
			if err := m.prefRepo.SetTheme(m.fingerprint, msg.Name); err != nil {
				log.Printf("Failed to save theme preference: %v", err)
			}
		}

		m.statusMsg = "Theme: " + msg.Theme.Name
		return m, tea.Batch(
			tea.ClearScreen,
			tea.Tick(1500*time.Millisecond, func(t time.Time) tea.Msg {
				return clearStatusMsg{}
			}),
		)

	case ThemeCancelledMsg:
		// Restore original theme
		originalTheme := m.themes[m.themeIndex]
		m.styles = theme.NewStyles(originalTheme, m.renderer)
		m.list.styles = m.styles
		m.reader.SetTheme(m.styles, m.themeNames[m.themeIndex])
		m.search.styles = m.styles
		m.themeSelector.SetStyles(m.styles)
		m.admin.SetStyles(m.styles)
		m.editor.SetStyles(m.styles)
		m.currentView = m.prevView
		m.prevView = m.savedPrevView
		return m, tea.ClearScreen

	case AdminCloseMsg:
		// Close admin view
		m.currentView = m.prevView
		m.prevView = m.savedPrevView
		// Refresh the list in case posts changed
		return m, tea.Batch(tea.ClearScreen, m.list.Init())

	case AdminNewPostMsg:
		// Launch editor for new post
		return m, m.launchEditorForNewPost()

	case AdminEditPostMsg:
		// Launch editor for existing post
		return m, m.launchEditorForPost(msg.Post)

	case EditorCloseMsg:
		// Editor closed - sync file and return to admin
		var statusCmd tea.Cmd
		if msg.Saved && msg.Err == nil {
			if syncErr := m.syncPost(msg.FilePath); syncErr != nil {
				log.Printf("Failed to sync post after edit: %v", syncErr)
			}
			m.statusMsg = "Saved: " + filepath.Base(msg.FilePath)
			statusCmd = tea.Tick(1500*time.Millisecond, func(t time.Time) tea.Msg {
				return clearStatusMsg{}
			})
		} else if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Save error: %v", msg.Err)
			m.isError = true
			statusCmd = tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
				return clearStatusMsg{}
			})
		}
		m.currentView = ViewAdmin
		return m, tea.Batch(tea.ClearScreen, m.admin.Init(), statusCmd)

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

	case ViewThemeSelector:
		var cmd tea.Cmd
		m.themeSelector, cmd = m.themeSelector.Update(msg)
		cmds = append(cmds, cmd)

	case ViewAdmin:
		var cmd tea.Cmd
		m.admin, cmd = m.admin.Update(msg)
		cmds = append(cmds, cmd)

	case ViewEditor:
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
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
	case ViewThemeSelector:
		content = m.themeSelector.View()
	case ViewAdmin:
		content = m.admin.View()
	case ViewEditor:
		content = m.editor.View()
	}

	// Build the full view with header and footer
	header := m.renderHeader()
	footer := m.renderFooter()

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		footer,
	)

	// Apply theme background to fill the entire terminal
	return m.styles.App.
		Width(m.width).
		Height(m.height).
		Render(view)
}

func (m *Model) renderHeader() string {
	// Use ASCII header if provided, otherwise just the title
	if m.config.ASCIIHeader != "" {
		return m.styles.Header.Width(m.width).Render(m.config.ASCIIHeader)
	}
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
	// Helper to render a single hint
	hint := func(key, desc string) string {
		return m.styles.HelpKey.Render(key) + m.styles.HelpDesc.Render(" "+desc)
	}

	// Common hints
	helpHint := hint("?", "help")
	searchHint := hint("/", "search")
	themeHint := hint("t", "theme")
	quitHint := hint("q", "quit")
	adminHint := hint("a", "admin")

	var hints []string

	switch m.currentView {
	case ViewHelp:
		hints = []string{hint("esc", "close"), themeHint, quitHint}
	case ViewReader:
		hints = []string{hint("esc", "back"), helpHint, searchHint, themeHint, quitHint}
		if m.isAdmin {
			hints = append(hints[:len(hints)-1], adminHint, quitHint)
		}
	case ViewSearch:
		hints = []string{hint("esc", "cancel")}
	case ViewThemeSelector:
		hints = []string{hint("↑/↓", "navigate"), hint("enter", "select"), hint("esc", "cancel")}
	case ViewAdmin:
		hints = []string{hint("n", "new"), hint("e", "edit"), hint("d", "delete"), hint("p", "toggle publish"), hint("esc", "back")}
	case ViewEditor:
		hints = []string{hint("ctrl+s", "save"), hint("ctrl+l", "line numbers"), hint("esc", "cancel")}
	default:
		hints = []string{helpHint, searchHint, themeHint, quitHint}
		if m.isAdmin {
			hints = append(hints[:len(hints)-1], adminHint, quitHint)
		}
	}

	separator := m.styles.HelpDesc.Render("  │  ")
	return strings.Join(hints, separator)
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
		m.renderHelpLine("enter/l", "Select/Open post"),
		m.renderHelpLine("esc/h", "Go back"),
		m.renderHelpLine("/", "Search posts"),
		m.renderHelpLine("t", "Cycle theme"),
		m.renderHelpLine("?", "Toggle this help"),
		m.renderHelpLine("q", "Quit"),
		"",
		m.styles.HelpSection.Render("Tips"),
		m.styles.HelpDesc.Render("Hold Shift + drag to select text"),
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return m.styles.Help.Render(content)
}

func (m *Model) renderHelpLine(key, desc string) string {
	k := m.styles.HelpKey.Render(key)
	d := m.styles.HelpDesc.Render(" - " + desc)
	return k + d
}

// clearStatusMsg is sent to clear the status message after a delay
type clearStatusMsg struct{}

// launchEditorForNewPost creates a new post file and opens it in the editor
func (m *Model) launchEditorForNewPost() tea.Cmd {
	title := fmt.Sprintf("New Post %s", time.Now().Format("2006-01-02-150405"))
	filePath, err := m.loader.CreatePost(title, m.config.Author)
	if err != nil {
		return func() tea.Msg {
			return StatusMsg{
				Message: fmt.Sprintf("Failed to create post: %v", err),
				IsError: true,
			}
		}
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	// Sync the new file to DB immediately
	if syncErr := m.syncPost(absPath); syncErr != nil {
		log.Printf("Failed to sync new post: %v", syncErr)
	}

	// Open in the in-TUI editor
	if openErr := m.editor.Open(absPath); openErr != nil {
		return func() tea.Msg {
			return StatusMsg{
				Message: fmt.Sprintf("Failed to open editor: %v", openErr),
				IsError: true,
			}
		}
	}
	m.editor.SetSize(m.width, m.height-2)
	m.currentView = ViewEditor
	return tea.ClearScreen
}

// launchEditorForPost opens an existing post in the in-TUI editor
func (m *Model) launchEditorForPost(post *storage.Post) tea.Cmd {
	if post == nil || post.Filepath == "" {
		return func() tea.Msg {
			return StatusMsg{
				Message: "No post selected or filepath missing",
				IsError: true,
			}
		}
	}

	absPath, err := filepath.Abs(post.Filepath)
	if err != nil {
		absPath = post.Filepath
	}

	if openErr := m.editor.Open(absPath); openErr != nil {
		return func() tea.Msg {
			return StatusMsg{
				Message: fmt.Sprintf("Failed to open editor: %v", openErr),
				IsError: true,
			}
		}
	}
	m.editor.SetSize(m.width, m.height-2)
	m.currentView = ViewEditor
	return tea.ClearScreen
}

// syncPost syncs a single post file to the database
func (m *Model) syncPost(filePath string) error {
	// Load the post from file
	post, err := m.loader.LoadPost(filePath)
	if err != nil {
		return fmt.Errorf("failed to load post: %w", err)
	}

	// Convert blog.Post to storage.Post
	status := storage.StatusDraft
	if !post.Draft {
		status = storage.StatusPublished
	}

	dbPost := &storage.Post{
		Slug:        post.Slug,
		Title:       post.Title,
		Filepath:    post.Filepath,
		Status:      status,
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   time.Now(),
		PublishedAt: post.PublishedAt,
		Tags:        post.Tags,
	}

	// Upsert to database
	if err := m.repo.UpsertBySlug(dbPost); err != nil {
		return err
	}

	// Update FTS search index
	tagsStr := strings.Join(post.Tags, " ")
	return m.repo.IndexPost(dbPost.ID, post.Title, tagsStr, post.Content)
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
