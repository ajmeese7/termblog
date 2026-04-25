package server

import (
	"strings"
	"testing"

	"github.com/ajmeese7/termblog/internal/app"
	"github.com/ajmeese7/termblog/internal/theme"
)

func draculaTheme() *theme.Theme {
	return theme.DefaultThemes()["dracula"]
}

func TestRenderFaviconSVG_LetterMode(t *testing.T) {
	cfg := app.FaviconConfig{
		Enabled: true,
		Mode:    app.FaviconModeLetter,
		Letter:  "T",
	}
	got := string(renderFaviconSVG(cfg, draculaTheme()))

	// Background rect uses theme background hex.
	if !strings.Contains(got, `fill="#282a36"`) {
		t.Errorf("letter mode missing background hex; body=%s", got)
	}
	// Text fill uses accent (dracula accent: #50fa7b).
	if !strings.Contains(got, `fill="#50fa7b"`) {
		t.Errorf("letter mode missing accent hex; body=%s", got)
	}
	if !strings.Contains(got, `>T</text>`) {
		t.Errorf("letter mode missing letter T; body=%s", got)
	}
}

func TestRenderFaviconSVG_LetterTruncatesToFirstRune(t *testing.T) {
	cfg := app.FaviconConfig{
		Enabled: true,
		Mode:    app.FaviconModeLetter,
		Letter:  "Hello",
	}
	got := string(renderFaviconSVG(cfg, draculaTheme()))
	if !strings.Contains(got, `>H</text>`) {
		t.Errorf("letter mode should truncate to first rune; body=%s", got)
	}
	if strings.Contains(got, "Hello") {
		t.Errorf("letter mode should not contain full string; body=%s", got)
	}
}

func TestRenderFaviconSVG_LetterFallsBackWhenEmpty(t *testing.T) {
	cfg := app.FaviconConfig{Enabled: true, Mode: app.FaviconModeLetter}
	got := string(renderFaviconSVG(cfg, draculaTheme()))
	if !strings.Contains(got, `>T</text>`) {
		t.Errorf("expected default letter T when Letter empty; body=%s", got)
	}
}

func TestRenderFaviconSVG_EmojiTransparent(t *testing.T) {
	cfg := app.FaviconConfig{
		Enabled: true,
		Mode:    app.FaviconModeEmoji,
		Emoji:   "📝",
		EmojiBg: app.FaviconEmojiBgTransparent,
	}
	got := string(renderFaviconSVG(cfg, draculaTheme()))

	if strings.Contains(got, `<rect`) {
		t.Errorf("emoji mode with transparent bg must not include <rect>; body=%s", got)
	}
	if !strings.Contains(got, "📝") {
		t.Errorf("emoji mode missing emoji; body=%s", got)
	}
}

func TestRenderFaviconSVG_EmojiThemed(t *testing.T) {
	cfg := app.FaviconConfig{
		Enabled: true,
		Mode:    app.FaviconModeEmoji,
		Emoji:   "📝",
		EmojiBg: app.FaviconEmojiBgThemed,
	}
	got := string(renderFaviconSVG(cfg, draculaTheme()))

	if !strings.Contains(got, `<rect width="32" height="32" fill="#282a36"/>`) {
		t.Errorf("emoji mode with themed bg must include themed <rect>; body=%s", got)
	}
}

func TestRenderFaviconSVG_EscapesXMLCharacters(t *testing.T) {
	cfg := app.FaviconConfig{
		Enabled: true,
		Mode:    app.FaviconModeEmoji,
		Emoji:   "<&>",
	}
	got := string(renderFaviconSVG(cfg, draculaTheme()))

	if strings.Contains(got, "<&>") {
		t.Errorf("XML-special characters must be escaped; body=%s", got)
	}
	if !strings.Contains(got, "&lt;") || !strings.Contains(got, "&amp;") || !strings.Contains(got, "&gt;") {
		t.Errorf("expected escaped < & > in output; body=%s", got)
	}
}

func TestRenderFaviconSVG_AccentFallsBackToPrimary(t *testing.T) {
	custom := &theme.Theme{
		Name: "Custom",
		Colors: theme.ThemeColors{
			Primary:    "#abcdef",
			Background: "#000000",
			// Accent intentionally empty
		},
	}
	cfg := app.FaviconConfig{Enabled: true, Mode: app.FaviconModeLetter, Letter: "X"}
	got := string(renderFaviconSVG(cfg, custom))

	if !strings.Contains(got, `fill="#abcdef"`) {
		t.Errorf("expected fallback to primary when accent empty; body=%s", got)
	}
}

func TestInjectFaviconHead_EnabledLetter(t *testing.T) {
	html := []byte(`<head><title>x</title><!--TERMBLOG_FAVICON--></head>`)
	cfg := app.FaviconConfig{Enabled: true, Mode: app.FaviconModeLetter}
	got := string(injectFaviconHead(html, cfg))

	if !strings.Contains(got, `<link rel="icon" href="/favicon">`) {
		t.Errorf("expected <link rel=icon> injected; got %s", got)
	}
	if !strings.Contains(got, `<meta name="termblog-favicon-mode" content="letter">`) {
		t.Errorf("expected mode meta tag injected; got %s", got)
	}
	if strings.Contains(got, "TERMBLOG_FAVICON") {
		t.Errorf("placeholder should be replaced; got %s", got)
	}
}

func TestInjectFaviconHead_Disabled(t *testing.T) {
	html := []byte(`<head><!--TERMBLOG_FAVICON--></head>`)
	cfg := app.FaviconConfig{Enabled: false, Mode: app.FaviconModeLetter}
	got := string(injectFaviconHead(html, cfg))

	if strings.Contains(got, "<link rel=\"icon\"") {
		t.Errorf("disabled favicon should not inject link; got %s", got)
	}
	if strings.Contains(got, "TERMBLOG_FAVICON") {
		t.Errorf("placeholder should still be removed when disabled; got %s", got)
	}
}
