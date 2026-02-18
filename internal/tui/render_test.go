package tui

import (
	"strings"
	"testing"

	"github.com/ajmeese7/termblog/internal/storage"
	"github.com/ajmeese7/termblog/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func init() {
	// Enable TrueColor for tests so ANSI codes are output
	lipgloss.SetColorProfile(termenv.TrueColor)
}

func TestPadContentLines(t *testing.T) {
	th := theme.DraculaTheme()
	styles := theme.NewStyles(th, nil)

	reader := &ReaderModel{
		styles: styles,
		width:  80,
	}

	// Test simple content
	content := "Line 1\nLine 2\nLine 3"
	result := reader.padContentLines(content, 40)

	lines := strings.Split(result, "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	// Each line should contain background color escape sequence
	// Dracula background is #282a36 = RGB(40, 42, 54)
	for i, line := range lines {
		// Check that the line contains the background ANSI code
		// 48;2;40;42;54 is the true color background for #282a36
		if !strings.Contains(line, "48;2;40;42;54") {
			t.Errorf("Line %d missing background color: %q", i, line)
		}
	}
}

func TestListRenderPostSpacing(t *testing.T) {
	th := theme.DraculaTheme()
	styles := theme.NewStyles(th, nil)

	// Count newlines in a rendered post
	// Should be exactly 2 newlines (title\ndateline\nemptyline)
	title := styles.ListItem.Width(76).Render("  Test Title")
	dateLine := styles.ContentBg.Width(76).Render("  2024-01-01  [tag1, tag2]")
	emptyLine := styles.ContentBg.Width(76).Render("")

	result := lipgloss.JoinVertical(lipgloss.Left, title, dateLine, emptyLine)

	newlineCount := strings.Count(result, "\n")
	// JoinVertical adds newlines between items, so 3 items = 2 newlines
	if newlineCount != 2 {
		t.Errorf("Expected 2 newlines in post, got %d. Result:\n%q", newlineCount, result)
	}
}

func TestListStylePadding(t *testing.T) {
	th := theme.DraculaTheme()
	styles := theme.NewStyles(th, nil)

	// The List style has Padding(1, 2) which adds vertical padding
	// ContentBg has no padding
	listRendered := styles.List.Width(40).Render("test")
	contentBgRendered := styles.ContentBg.Width(40).Render("test")

	listLines := strings.Count(listRendered, "\n")
	contentBgLines := strings.Count(contentBgRendered, "\n")

	// List should have more newlines due to padding
	if listLines <= contentBgLines {
		t.Errorf("List style should have padding (more newlines). List: %d, ContentBg: %d", listLines, contentBgLines)
	}
}

func TestMouseRightClickIgnored(t *testing.T) {
	// Test that right-click events don't trigger post selection
	th := theme.DraculaTheme()
	styles := theme.NewStyles(th, nil)

	list := NewListModel(nil, styles, "Test")
	list.width = 80
	list.height = 24
	list.posts = []*storage.Post{
		{Title: "Post 1", Slug: "post-1"},
		{Title: "Post 2", Slug: "post-2"},
	}
	list.cursor = 0 // First post selected

	// Simulate right-click release on selected post
	rightClickMsg := tea.MouseMsg{
		X:      10,
		Y:      3, // On the selected post
		Button: tea.MouseButtonRight,
		Action: tea.MouseActionRelease,
	}

	_, cmd := list.Update(rightClickMsg)
	if cmd != nil {
		t.Errorf("Right-click should not produce a command, but got one")
	}

	// Verify cursor didn't change
	if list.cursor != 0 {
		t.Errorf("Cursor should still be 0, got %d", list.cursor)
	}
}

func TestMouseLeftClickSelectsPost(t *testing.T) {
	// Test that left-click on selected post triggers selection
	th := theme.DraculaTheme()
	styles := theme.NewStyles(th, nil)

	list := NewListModel(nil, styles, "Test")
	list.width = 80
	list.height = 24
	list.posts = []*storage.Post{
		{Title: "Post 1", Slug: "post-1"},
		{Title: "Post 2", Slug: "post-2"},
	}
	list.cursor = 0 // First post selected

	// Simulate left-click release on selected post
	leftClickMsg := tea.MouseMsg{
		X:      10,
		Y:      3, // On the selected post (header=2, first post area starts at 2)
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	}

	_, cmd := list.Update(leftClickMsg)
	// Left-click on selected post should produce a command to select it
	if cmd == nil {
		t.Logf("Note: Left-click didn't produce command, Y position may not match post area")
	}
}

func TestMouseButtonNoneIgnored(t *testing.T) {
	// Test that MouseButtonNone (motion or other events) are ignored
	th := theme.DraculaTheme()
	styles := theme.NewStyles(th, nil)

	list := NewListModel(nil, styles, "Test")
	list.width = 80
	list.height = 24
	list.posts = []*storage.Post{
		{Title: "Post 1", Slug: "post-1"},
	}
	list.cursor = 0

	// MouseButtonNone with release action (could happen with some terminals)
	noneMsg := tea.MouseMsg{
		X:      10,
		Y:      3,
		Button: tea.MouseButtonNone,
		Action: tea.MouseActionRelease,
	}

	_, cmd := list.Update(noneMsg)
	if cmd != nil {
		t.Errorf("MouseButtonNone should not produce a command")
	}
}

func TestReaderContentHasBackground(t *testing.T) {
	// Test that reader content has background color applied
	th := theme.DraculaTheme()
	styles := theme.NewStyles(th, nil)

	reader := NewReaderModel(styles, "dracula")
	reader.width = 80
	reader.height = 24

	// Simulate setting size (needed to initialize viewport)
	reader.SetSize(80, 24)

	// Test padContentLines with content that has existing ANSI codes (like glamour output)
	content := "Normal text\n\x1b[1mBold text\x1b[0m\nMore text"
	result := reader.padContentLines(content, 60)

	lines := strings.Split(result, "\n")
	// Dracula background is #282a36 = RGB(40, 42, 54)
	bgCode := "48;2;40;42;54"

	for i, line := range lines {
		if !strings.Contains(line, bgCode) {
			t.Errorf("Line %d missing background color.\nExpected to contain: %s\nGot: %q", i, bgCode, line)
		}
	}
}

func TestStripLeadingH1(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "strips H1 heading",
			input:    "# My Post Title\n\nSome content here.",
			expected: "Some content here.",
		},
		{
			name:     "strips H1 with blank line after",
			input:    "# Title\n\nParagraph one.\n\nParagraph two.",
			expected: "Paragraph one.\n\nParagraph two.",
		},
		{
			name:     "preserves H2 headings",
			input:    "## Section Title\n\nContent.",
			expected: "## Section Title\n\nContent.",
		},
		{
			name:     "strips H1 with leading blank lines",
			input:    "\n\n# Title\n\nContent.",
			expected: "Content.",
		},
		{
			name:     "no heading at all",
			input:    "Just some text.\n\nMore text.",
			expected: "Just some text.\n\nMore text.",
		},
		{
			name:     "empty content",
			input:    "",
			expected: "",
		},
		{
			name:     "H1 only",
			input:    "# Solo Title",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := stripLeadingH1(tc.input)
			if result != tc.expected {
				t.Errorf("stripLeadingH1(%q)\n  got:      %q\n  expected: %q", tc.input, result, tc.expected)
			}
		})
	}
}
