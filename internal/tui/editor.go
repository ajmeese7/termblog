package tui

import (
	"os"
	"strings"

	"github.com/ajmeese7/termblog/internal/theme"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// EditorModel provides an in-TUI text editor for post files
type EditorModel struct {
	textarea textarea.Model
	styles   *theme.Styles
	filePath string

	width  int
	height int
}

// NewEditorModel creates a new editor model
func NewEditorModel(styles *theme.Styles) *EditorModel {
	ta := textarea.New()
	ta.ShowLineNumbers = true
	ta.CharLimit = 0 // No character limit
	ta.MaxHeight = 0 // No line limit

	return &EditorModel{
		textarea: ta,
		styles:   styles,
	}
}

// SetSize updates the editor dimensions
func (m *EditorModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.textarea.SetWidth(width - 2)
	m.textarea.SetHeight(height - 4) // Room for header + help line
}

// SetStyles updates the styles
func (m *EditorModel) SetStyles(styles *theme.Styles) {
	m.styles = styles
}

// Open loads a file into the editor
func (m *EditorModel) Open(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	m.filePath = filePath
	m.textarea.SetValue(string(content))
	m.textarea.Focus()
	m.textarea.CursorStart()
	return nil
}

// Update handles input
func (m *EditorModel) Update(msg tea.Msg) (*EditorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			// Save the file
			err := os.WriteFile(m.filePath, []byte(m.textarea.Value()), 0644)
			if err != nil {
				return m, func() tea.Msg {
					return EditorCloseMsg{FilePath: m.filePath, Err: err}
				}
			}
			return m, func() tea.Msg {
				return EditorCloseMsg{FilePath: m.filePath, Saved: true}
			}
		case "esc", "ctrl+c":
			return m, func() tea.Msg {
				return EditorCloseMsg{FilePath: m.filePath}
			}
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// View renders the editor
func (m *EditorModel) View() string {
	header := m.styles.Title.Render("Editing: " + m.filePath)

	content := header + "\n" + m.textarea.View()

	// Fix background on each line
	lines := strings.Split(content, "\n")
	bgCode := extractBgCode(m.styles.ContentBg)
	for i, line := range lines {
		if bgCode != "" {
			line = strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+bgCode)
		}
		lines[i] = m.styles.ContentBg.Width(m.width).Render(line)
	}
	return strings.Join(lines, "\n")
}

// EditorCloseMsg is sent when the editor is closed
type EditorCloseMsg struct {
	FilePath string
	Saved    bool
	Err      error
}
