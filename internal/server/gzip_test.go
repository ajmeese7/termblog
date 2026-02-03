package server

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGzipMiddleware_CompressesWhenAccepted(t *testing.T) {
	handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body>Hello, World!</body></html>"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Error("Expected Content-Encoding: gzip header")
	}
	if rec.Header().Get("Vary") != "Accept-Encoding" {
		t.Error("Expected Vary: Accept-Encoding header")
	}

	// Verify the response is actually gzipped
	gr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	body, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("Failed to read gzipped body: %v", err)
	}

	if !strings.Contains(string(body), "Hello, World!") {
		t.Error("Expected body to contain 'Hello, World!'")
	}
}

func TestGzipMiddleware_NoCompressWithoutHeader(t *testing.T) {
	handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><body>Hello!</body></html>"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	// No Accept-Encoding header
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Should not compress without Accept-Encoding header")
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Hello!") {
		t.Error("Expected uncompressed body")
	}
}

func TestGzipMiddleware_SkipsWebSocket(t *testing.T) {
	handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("websocket response"))
	}))

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Should not compress WebSocket upgrade requests")
	}
}

func TestIsCompressible(t *testing.T) {
	tests := []struct {
		contentType  string
		compressible bool
	}{
		{"text/html", true},
		{"text/html; charset=utf-8", true},
		{"text/css", true},
		{"text/plain", true},
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"application/rss+xml", true},
		{"application/feed+json", true},
		{"image/png", false},
		{"image/jpeg", false},
		{"application/octet-stream", false},
		{"video/mp4", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := isCompressible(tt.contentType)
			if result != tt.compressible {
				t.Errorf("isCompressible(%q) = %v, want %v", tt.contentType, result, tt.compressible)
			}
		})
	}
}
