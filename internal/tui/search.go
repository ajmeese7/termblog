package tui

import (
	"fmt"
	"strings"

	"github.com/ajmeese7/termblog/internal/blog"
	"github.com/ajmeese7/termblog/internal/storage"
	"github.com/ajmeese7/termblog/internal/theme"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SearchModel handles the search view
type SearchModel struct {
	repo   *storage.PostRepository
	loader *blog.ContentLoader
	styles *theme.Styles
	keyMap KeyMap

	input   textinput.Model
	results []*storage.Post
	cursor  int

	width  int
	height int

	searching bool
	err       error
}

// NewSearchModel creates a new search model
func NewSearchModel(repo *storage.PostRepository, loader *blog.ContentLoader, styles *theme.Styles) *SearchModel {
	input := textinput.New()
	input.Placeholder = "Search posts..."
	input.CharLimit = 100
	input.Width = 40

	return &SearchModel{
		repo:   repo,
		loader: loader,
		styles: styles,
		keyMap: DefaultKeyMap(),
		input:  input,
	}
}

// SetSize updates the dimensions
func (m *SearchModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.input.Width = width - 10
}

// Focus activates the search input
func (m *SearchModel) Focus() tea.Cmd {
	m.input.SetValue("")
	m.results = nil
	m.cursor = 0
	return m.input.Focus()
}

// Update implements tea.Model
func (m *SearchModel) Update(msg tea.Msg) (*SearchModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case searchResultsMsg:
		m.searching = false
		m.results = msg.results
		m.cursor = 0
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			m.input.Blur()
			return m, func() tea.Msg { return SearchCancelledMsg{} }

		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if len(m.results) > 0 && m.cursor < len(m.results) {
				// Select a result
				return m, m.selectResult()
			} else if m.input.Focused() && m.input.Value() != "" {
				// Perform search
				return m, m.performSearch()
			}

		case key.Matches(msg, m.keyMap.Up):
			if !m.input.Focused() || len(m.results) > 0 {
				m.cursor--
				if m.cursor < 0 {
					m.cursor = 0
				}
			}

		case key.Matches(msg, m.keyMap.Down):
			if !m.input.Focused() || len(m.results) > 0 {
				m.cursor++
				if m.cursor >= len(m.results) {
					m.cursor = len(m.results) - 1
				}
				if m.cursor < 0 {
					m.cursor = 0
				}
			}

		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			// Toggle between input and results
			if m.input.Focused() && len(m.results) > 0 {
				m.input.Blur()
			} else {
				m.input.Focus()
			}

		default:
			// Update text input
			if m.input.Focused() {
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				cmds = append(cmds, cmd)

				// Auto-search after typing
				if m.input.Value() != "" {
					cmds = append(cmds, m.performSearch())
				} else {
					m.results = nil
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m *SearchModel) View() string {
	var sections []string

	// Search input
	inputStyle := m.styles.SearchInput.Width(m.width - 6)
	sections = append(sections, inputStyle.Render(m.input.View()))

	// Search hint
	hint := m.styles.SearchHint.Render("Press Enter to search, Tab to navigate results, Esc to cancel")
	sections = append(sections, hint)
	sections = append(sections, "")

	// Error message
	if m.err != nil {
		sections = append(sections, m.styles.StatusError.Render(fmt.Sprintf("Error: %v", m.err)))
		sections = append(sections, "")
	}

	// Results
	if m.searching {
		sections = append(sections, m.styles.HelpDesc.Render("Searching..."))
	} else if len(m.results) == 0 && m.input.Value() != "" {
		sections = append(sections, m.styles.HelpDesc.Render("No results found"))
	} else if len(m.results) > 0 {
		sections = append(sections, m.styles.HelpSection.Render(fmt.Sprintf("Results (%d)", len(m.results))))
		sections = append(sections, "")

		for i, post := range m.results {
			if i >= 10 {
				sections = append(sections, m.styles.HelpDesc.Render(fmt.Sprintf("... and %d more", len(m.results)-10)))
				break
			}
			sections = append(sections, m.renderResult(i, post))
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return m.styles.Search.Width(m.width).Render(content)
}

func (m *SearchModel) renderResult(idx int, post *storage.Post) string {
	isSelected := idx == m.cursor && !m.input.Focused()

	// Format date
	var dateStr string
	if post.PublishedAt != nil {
		dateStr = post.PublishedAt.Format("2006-01-02")
	} else {
		dateStr = post.CreatedAt.Format("2006-01-02")
	}

	// Format title
	var title string
	if isSelected {
		title = m.styles.ListSelected.Render("► " + post.Title)
	} else {
		title = m.styles.ListItem.Render("  " + post.Title)
	}

	// Format tags
	var tagsStr string
	if len(post.Tags) > 0 {
		tagsStr = " [" + strings.Join(post.Tags, ", ") + "]"
	}

	date := m.styles.ListDate.Render("  " + dateStr + tagsStr)

	return lipgloss.JoinVertical(lipgloss.Left, title, date)
}

func (m *SearchModel) performSearch() tea.Cmd {
	query := m.input.Value()
	if query == "" {
		return nil
	}

	m.searching = true

	return func() tea.Msg {
		results, err := m.repo.Search(query, 20)
		return searchResultsMsg{
			results: results,
			err:     err,
		}
	}
}

func (m *SearchModel) selectResult() tea.Cmd {
	if m.cursor >= len(m.results) {
		return nil
	}

	post := m.results[m.cursor]

	return func() tea.Msg {
		content, err := loadPostContent(post.Filepath)
		if err != nil {
			return StatusMsg{
				Message: fmt.Sprintf("Failed to load post: %v", err),
				IsError: true,
			}
		}

		return SearchCompletedMsg{
			SelectedPost: post,
			Content:      content,
		}
	}
}

// Messages

type searchResultsMsg struct {
	results []*storage.Post
	err     error
}
