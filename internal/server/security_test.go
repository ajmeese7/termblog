package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeadersMiddlewareSetsCSPNonce(t *testing.T) {
	var seenNonces []string
	s := &HTTPServer{}
	handler := s.securityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenNonces = append(seenNonces, nonceFromContext(r.Context()))
	}))

	csp1 := callAndGetCSP(t, handler)
	csp2 := callAndGetCSP(t, handler)

	for _, csp := range []string{csp1, csp2} {
		if !strings.Contains(csp, "'wasm-unsafe-eval'") {
			t.Errorf("CSP missing wasm-unsafe-eval: %q", csp)
		}
		if !strings.Contains(csp, "'nonce-") {
			t.Errorf("CSP missing nonce: %q", csp)
		}
	}

	if len(seenNonces) != 2 {
		t.Fatalf("expected 2 nonces in context, got %d", len(seenNonces))
	}
	if seenNonces[0] == "" {
		t.Error("nonce missing from request context")
	}
	if seenNonces[0] == seenNonces[1] {
		t.Error("nonces should differ between requests")
	}
	if !strings.Contains(csp1, "'nonce-"+seenNonces[0]+"'") {
		t.Errorf("CSP nonce header %q does not match context nonce %q", csp1, seenNonces[0])
	}
}

func TestHandleIndexInjectsNonceIntoScriptTags(t *testing.T) {
	s := &HTTPServer{
		indexHTML: []byte(`<!doctype html><html><head>` +
			`<script>var a=1;</script>` +
			`<script type="module" src="/app.js"></script>` +
			`</head><body></body></html>`),
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), cspNonceKey, "test-nonce-abc"))
	rec := httptest.NewRecorder()

	s.handleIndex(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if got := resp.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Errorf("Content-Type = %q, want text/html prefix", got)
	}

	wantInline := `<script nonce="test-nonce-abc">var a=1;</script>`
	wantModule := `<script nonce="test-nonce-abc" type="module" src="/app.js">`
	if !strings.Contains(bodyStr, wantInline) {
		t.Errorf("inline script not nonce-tagged. body=%s", bodyStr)
	}
	if !strings.Contains(bodyStr, wantModule) {
		t.Errorf("module script not nonce-tagged. body=%s", bodyStr)
	}
	if strings.Contains(bodyStr, `<script>`) || strings.Contains(bodyStr, `<script type="module"`) && !strings.Contains(bodyStr, `<script nonce=`) {
		t.Errorf("found a <script tag without a nonce. body=%s", bodyStr)
	}
}

func callAndGetCSP(t *testing.T, h http.Handler) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Header().Get("Content-Security-Policy")
}
