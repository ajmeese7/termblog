package tui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/ajmeese7/termblog/internal/theme/styles"
	"github.com/charmbracelet/glamour"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func renderTOCForTest(t *testing.T, markdown string) string {
	t.Helper()

	styleJSON, err := styles.GetStyle("dracula")
	if err != nil {
		t.Fatalf("failed to get style: %v", err)
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes(styleJSON),
		glamour.WithWordWrap(200),
	)
	if err != nil {
		t.Fatalf("failed to create renderer: %v", err)
	}

	out, err := renderer.Render(markdown)
	if err != nil {
		t.Fatalf("failed to render markdown: %v", err)
	}

	out = ansiRegex.ReplaceAllString(out, "")
	out = strings.ReplaceAll(out, "\u00a0", " ")
	return out
}

func TestGenerateTOCRenderedNumberedHeadingsRemainBulletLines(t *testing.T) {
	content := `## 1. Server baseline
## 2. Configure production values
## 3. Run as a service
`

	toc := generateTOC(content)
	rendered := renderTOCForTest(t, toc)

	if strings.Contains(rendered, `\.`) {
		t.Fatalf("did not expect escaped dots in rendered TOC, got:\n%s", rendered)
	}

	for _, heading := range []string{
		"1. Server baseline",
		"2. Configure production values",
		"3. Run as a service",
	} {
		linePattern := regexp.MustCompile(`(?m)^\s*•\s+` + regexp.QuoteMeta(heading) + `\s*$`)
		if !linePattern.MatchString(rendered) {
			t.Fatalf("expected rendered TOC to contain bullet line for %q, got:\n%s", heading, rendered)
		}
	}
}

func TestGenerateTOCLeavesNormalHeadingsReadable(t *testing.T) {
	content := `## Introduction
## Deployment`

	toc := generateTOC(content)
	rendered := renderTOCForTest(t, toc)

	for _, heading := range []string{"Introduction", "Deployment"} {
		linePattern := regexp.MustCompile(`(?m)^\s*•\s+` + regexp.QuoteMeta(heading) + `\s*$`)
		if !linePattern.MatchString(rendered) {
			t.Fatalf("expected rendered TOC to contain bullet line for %q, got:\n%s", heading, rendered)
		}
	}
}
