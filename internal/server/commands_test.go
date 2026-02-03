package server

import (
	"bytes"
	"strings"
	"testing"
)

func TestCommandHandler_HandleHelp(t *testing.T) {
	handler := NewCommandHandler(nil, nil, nil)
	buf := &bytes.Buffer{}

	handled, err := handler.HandleCommand(buf, []string{"help"})
	if err != nil {
		t.Errorf("help command returned error: %v", err)
	}
	if !handled {
		t.Error("help command should be handled")
	}

	output := buf.String()
	if !strings.Contains(output, "TermBlog SSH Commands") {
		t.Error("help output should contain title")
	}
	if !strings.Contains(output, "posts") {
		t.Error("help output should list posts command")
	}
	if !strings.Contains(output, "read") {
		t.Error("help output should list read command")
	}
	if !strings.Contains(output, "rss") {
		t.Error("help output should list rss command")
	}
	if !strings.Contains(output, "search") {
		t.Error("help output should list search command")
	}
}

func TestCommandHandler_UnknownCommand(t *testing.T) {
	handler := NewCommandHandler(nil, nil, nil)
	buf := &bytes.Buffer{}

	handled, err := handler.HandleCommand(buf, []string{"unknown"})
	if err != nil {
		t.Errorf("unknown command returned error: %v", err)
	}
	if handled {
		t.Error("unknown command should not be handled")
	}
}

func TestCommandHandler_NoCommand(t *testing.T) {
	handler := NewCommandHandler(nil, nil, nil)
	buf := &bytes.Buffer{}

	handled, err := handler.HandleCommand(buf, []string{})
	if err != nil {
		t.Errorf("no command returned error: %v", err)
	}
	if handled {
		t.Error("no command should not be handled (should launch TUI)")
	}
}

func TestCommandHandler_ReadNoArgs(t *testing.T) {
	handler := NewCommandHandler(nil, nil, nil)
	buf := &bytes.Buffer{}

	handled, err := handler.HandleCommand(buf, []string{"read"})
	if err == nil {
		t.Error("read with no args should return error")
	}
	if !handled {
		t.Error("read command should be handled even with error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Error("error should contain usage information")
	}
}

func TestCommandHandler_SearchNoArgs(t *testing.T) {
	handler := NewCommandHandler(nil, nil, nil)
	buf := &bytes.Buffer{}

	handled, err := handler.HandleCommand(buf, []string{"search"})
	if err == nil {
		t.Error("search with no args should return error")
	}
	if !handled {
		t.Error("search command should be handled even with error")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Error("error should contain usage information")
	}
}

func TestRenderPlainText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "heading",
			input:    "# Hello World",
			expected: "Hello World",
		},
		{
			name:     "bold",
			input:    "This is **bold** text",
			expected: "This is bold text",
		},
		{
			name:     "italic",
			input:    "This is *italic* text",
			expected: "This is italic text",
		},
		{
			name:     "inline code",
			input:    "This is `code` text",
			expected: "This is code text",
		},
		{
			name:     "link",
			input:    "Check [this link](https://example.com) out",
			expected: "Check this link out",
		},
		{
			name:     "multiple formatting",
			input:    "## **Bold** heading with `code`",
			expected: "Bold heading with code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderPlainText(tt.input)
			result = strings.TrimSpace(result)
			if result != tt.expected {
				t.Errorf("renderPlainText(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCommandHandler_HelpAliases(t *testing.T) {
	handler := NewCommandHandler(nil, nil, nil)

	// Test that all aliases work for help command (doesn't need repo)
	helpAliases := []string{"help", "-h", "--help"}
	for _, alias := range helpAliases {
		buf := &bytes.Buffer{}
		handled, err := handler.HandleCommand(buf, []string{alias})
		if err != nil {
			t.Errorf("alias %q returned error: %v", alias, err)
		}
		if !handled {
			t.Errorf("alias %q should be handled", alias)
		}
	}
}

func TestCommandHandler_CaseSensitivity(t *testing.T) {
	handler := NewCommandHandler(nil, nil, nil)

	// Test that commands are case-insensitive
	buf := &bytes.Buffer{}
	handled, err := handler.HandleCommand(buf, []string{"HELP"})
	if err != nil {
		t.Errorf("HELP command returned error: %v", err)
	}
	if !handled {
		t.Error("HELP (uppercase) should be handled")
	}

	buf.Reset()
	handled, err = handler.HandleCommand(buf, []string{"Help"})
	if err != nil {
		t.Errorf("Help command returned error: %v", err)
	}
	if !handled {
		t.Error("Help (mixed case) should be handled")
	}
}
