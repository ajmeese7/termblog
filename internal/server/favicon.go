package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ajmeese7/termblog/internal/app"
	"github.com/ajmeese7/termblog/internal/theme"
)

// faviconHeadPlaceholder is replaced once at server startup with the
// theme-aware <link>/<meta> tags. Lives in wasm_dist/index.html so the
// substitution is a single byte-level replace.
const faviconHeadPlaceholder = "<!--TERMBLOG_FAVICON-->"

// prepareFavicon pre-renders SVGs for letter/emoji modes (one per known
// theme) and computes the extra img-src origin for image-URL mode. For local
// image paths and disabled mode there is nothing to pre-compute.
func (s *HTTPServer) prepareFavicon() error {
	if !s.faviconCfg.Enabled {
		return nil
	}

	switch s.faviconCfg.Mode {
	case app.FaviconModeLetter, app.FaviconModeEmoji:
		s.faviconRendered = make(map[string]*faviconResource, len(theme.DefaultThemes()))
		for key, t := range theme.DefaultThemes() {
			body := renderFaviconSVG(s.faviconCfg, t)
			s.faviconRendered[key] = &faviconResource{
				body: body,
				etag: faviconETag(body),
			}
		}
	case app.FaviconModeImage:
		// URL form needs its origin allowlisted in CSP so the cross-origin
		// fetch from <link rel="icon"> actually completes.
		if strings.HasPrefix(s.faviconCfg.Image, "http://") || strings.HasPrefix(s.faviconCfg.Image, "https://") {
			u, err := url.Parse(s.faviconCfg.Image)
			if err != nil {
				return fmt.Errorf("parse favicon URL: %w", err)
			}
			origin := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
			s.faviconImageOrigin = origin
			s.faviconExtraImgSrc = " " + origin
		}
	}
	return nil
}

// handleFavicon serves either a pre-rendered SVG (letter/emoji modes), a local
// image file (image mode + local path), or a redirect (image mode + URL).
func (s *HTTPServer) handleFavicon(w http.ResponseWriter, r *http.Request) {
	if !s.faviconCfg.Enabled {
		http.NotFound(w, r)
		return
	}

	switch s.faviconCfg.Mode {
	case app.FaviconModeImage:
		if s.faviconCfg.ResolvedImagePath != "" {
			http.ServeFile(w, r, s.faviconCfg.ResolvedImagePath)
			return
		}
		http.Redirect(w, r, s.faviconCfg.Image, http.StatusFound)
		return
	}

	// Letter / emoji: pick by query, fall back to the configured default.
	key := r.URL.Query().Get("theme")
	res, ok := s.faviconRendered[key]
	if !ok {
		res = s.faviconRendered[s.themeKey]
	}
	if res == nil {
		http.NotFound(w, r)
		return
	}

	if match := r.Header.Get("If-None-Match"); match != "" && match == res.etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("ETag", res.etag)
	w.Write(res.body)
}

// injectFaviconHead substitutes the favicon placeholder in the WASM index
// HTML with a theme-aware <link>/<meta> block. When the feature is disabled
// the placeholder collapses to nothing and the browser falls through to its
// default /favicon.ico request (which we 404).
func injectFaviconHead(indexHTML []byte, cfg app.FaviconConfig) []byte {
	var replacement string
	if cfg.Enabled {
		replacement = fmt.Sprintf(
			`<link rel="icon" href="/favicon"><meta name="termblog-favicon-mode" content="%s">`,
			xmlAttr(cfg.Mode),
		)
	}
	return bytes.Replace(indexHTML, []byte(faviconHeadPlaceholder), []byte(replacement), 1)
}

// faviconETag returns a strong ETag derived from the favicon body. The hash
// is short — collisions don't break correctness, only revalidation efficiency.
func faviconETag(body []byte) string {
	sum := sha256.Sum256(body)
	return fmt.Sprintf(`"%x"`, sum[:8])
}

// renderFaviconSVG returns the bytes of a 32x32 SVG favicon rendered from the
// supplied favicon config and theme. Mode must be "letter" or "emoji"; image
// mode does not flow through this function.
//
// The SVG is intentionally tiny: a square viewBox, an optional background
// rect, and one centered <text> element. No fancy layout. The font stack is
// the same one browsers use for monospace defaults so the letter looks correct
// without any web-font fetch.
func renderFaviconSVG(cfg app.FaviconConfig, t *theme.Theme) []byte {
	switch cfg.Mode {
	case app.FaviconModeEmoji:
		return renderEmojiFavicon(cfg, t)
	default:
		// Letter mode is the safe fallback for unknown modes — image mode
		// never reaches here.
		return renderLetterFavicon(cfg, t)
	}
}

// renderLetterFavicon draws the configured letter, recolored to the theme's
// accent (falling back to primary when the theme leaves accent empty), on top
// of the theme's background fill.
func renderLetterFavicon(cfg app.FaviconConfig, t *theme.Theme) []byte {
	bg := nonEmpty(t.Colors.Background, "#000000")
	fg := nonEmpty(t.Colors.Accent, t.Colors.Primary, "#ffffff")
	letter := firstRune(cfg.Letter, 'T')

	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32">`)
	fmt.Fprintf(&b, `<rect width="32" height="32" fill="%s"/>`, xmlAttr(bg))
	fmt.Fprintf(
		&b,
		`<text x="16" y="22" text-anchor="middle" font-family="ui-monospace, SFMono-Regular, Menlo, Consolas, monospace" font-weight="700" font-size="22" fill="%s">%s</text>`,
		xmlAttr(fg),
		xmlText(string(letter)),
	)
	b.WriteString(`</svg>`)
	return []byte(b.String())
}

// renderEmojiFavicon centers the configured emoji glyph in the viewBox. The
// background is either omitted (transparent) or filled with the theme
// background, depending on cfg.EmojiBg.
func renderEmojiFavicon(cfg app.FaviconConfig, t *theme.Theme) []byte {
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32">`)
	if cfg.EmojiBg == app.FaviconEmojiBgThemed {
		bg := nonEmpty(t.Colors.Background, "#000000")
		fmt.Fprintf(&b, `<rect width="32" height="32" fill="%s"/>`, xmlAttr(bg))
	}
	emoji := cfg.Emoji
	if emoji == "" {
		emoji = "📝"
	}
	fmt.Fprintf(
		&b,
		`<text x="16" y="25" text-anchor="middle" font-size="26">%s</text>`,
		xmlText(emoji),
	)
	b.WriteString(`</svg>`)
	return []byte(b.String())
}

// xmlAttr escapes a string for safe inclusion as an XML attribute value.
func xmlAttr(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// xmlText escapes a string for safe inclusion as XML character data.
func xmlText(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// nonEmpty returns the first non-empty argument, or "" if all are empty.
func nonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// firstRune returns the first rune of s, or fallback when s is empty.
func firstRune(s string, fallback rune) rune {
	for _, r := range s {
		return r
	}
	return fallback
}
