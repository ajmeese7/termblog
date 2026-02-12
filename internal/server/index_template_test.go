package server

import (
	"strings"
	"testing"
)

func TestIndexTemplateUsesProtocolAwareWebSocketURL(t *testing.T) {
	b, err := templatesFS.ReadFile("templates/index.html")
	if err != nil {
		t.Fatalf("failed to read index template: %v", err)
	}

	body := string(b)

	if strings.Contains(body, "{{.WSUrl}}") {
		t.Fatal("index template should not depend on server-rendered WSUrl")
	}

	if !strings.Contains(body, "window.location.protocol === \"https:\" ? \"wss\" : \"ws\"") {
		t.Fatal("index template should build a protocol-aware ws/wss URL")
	}
}
