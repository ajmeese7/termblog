package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ajmeese7/termblog/internal/theme"
	"github.com/ajmeese7/termblog/internal/theme/styles"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestGlamourResetCodesGetBackgroundRestored(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	th := theme.DraculaTheme()
	themeStyles := theme.NewStyles(th, nil)

	content := "# Hello World\n\nThis is a test paragraph.\n\n- Item 1\n- Item 2"
	contentWidth := 60

	styleJSON, err := styles.GetStyle("dracula")
	if err != nil {
		t.Fatalf("Failed to get style: %v", err)
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes(styleJSON),
		glamour.WithWordWrap(contentWidth),
	)
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	reader := &ReaderModel{
		styles: themeStyles,
		width:  80,
	}
	padded := reader.padContentLines(rendered, contentWidth)

	bgCode := "48;2;40;42;54" // Dracula background RGB
	resetCount := strings.Count(padded, "\x1b[0m")
	resetWithBgCount := strings.Count(padded, "\x1b[0m\x1b[48;2;40;42;54m")

	if resetCount == 0 {
		t.Fatal("No ANSI reset codes found in glamour output")
	}

	// Most resets should be followed by background restoration
	if resetWithBgCount < resetCount/2 {
		t.Errorf("Only %d/%d resets followed by background code", resetWithBgCount, resetCount)
	}

	if !strings.Contains(padded, bgCode) {
		t.Errorf("Background code %s not found in output", bgCode)
	}
}

func TestTrueColorRendering(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	th := theme.DraculaTheme()
	themeStyles := theme.NewStyles(th, nil)

	result := themeStyles.ContentBg.Width(40).Render("test")

	if !strings.Contains(result, "48;2;") {
		t.Errorf("TrueColor background not in output: %q", result)
	}
}

func TestWebThemeOSCFormat(t *testing.T) {
	themeName := "dracula"
	osc := fmt.Sprintf("\x1b]7777;theme=%s\x07", themeName)

	if len(osc) != 21 {
		t.Errorf("Expected OSC length 21, got %d", len(osc))
	}

	if !strings.Contains(osc, "7777;theme=dracula") {
		t.Errorf("OSC sequence malformed: %q", osc)
	}

	if osc[0] != '\x1b' || osc[len(osc)-1] != '\x07' {
		t.Error("OSC missing ESC prefix or BEL terminator")
	}
}
