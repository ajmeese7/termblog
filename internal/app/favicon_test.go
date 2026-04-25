package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateFavicon_DisabledSkipsAllChecks(t *testing.T) {
	cfg := FaviconConfig{Enabled: false, Mode: "image"} // image without path would normally fail
	if err := validateAndResolveFavicon(&cfg, t.TempDir()); err != nil {
		t.Fatalf("disabled favicon should skip validation: %v", err)
	}
}

func TestValidateFavicon_DefaultsApplied(t *testing.T) {
	cfg := FaviconConfig{Enabled: true}
	if err := validateAndResolveFavicon(&cfg, t.TempDir()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Mode != FaviconModeLetter {
		t.Errorf("Mode = %q, want letter", cfg.Mode)
	}
	if cfg.Letter != "T" {
		t.Errorf("Letter = %q, want T", cfg.Letter)
	}
	if cfg.Emoji != "📝" {
		t.Errorf("Emoji = %q, want 📝", cfg.Emoji)
	}
	if cfg.EmojiBg != FaviconEmojiBgTransparent {
		t.Errorf("EmojiBg = %q, want transparent", cfg.EmojiBg)
	}
}

func TestValidateFavicon_RejectsUnknownMode(t *testing.T) {
	cfg := FaviconConfig{Enabled: true, Mode: "rainbow"}
	err := validateAndResolveFavicon(&cfg, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "favicon.mode") {
		t.Errorf("expected mode error, got %v", err)
	}
}

func TestValidateFavicon_RejectsUnknownEmojiBg(t *testing.T) {
	cfg := FaviconConfig{Enabled: true, Mode: FaviconModeEmoji, EmojiBg: "neon"}
	err := validateAndResolveFavicon(&cfg, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "emoji_bg") {
		t.Errorf("expected emoji_bg error, got %v", err)
	}
}

func TestValidateFavicon_ImageModeRequiresImage(t *testing.T) {
	cfg := FaviconConfig{Enabled: true, Mode: FaviconModeImage}
	err := validateAndResolveFavicon(&cfg, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "image is empty") {
		t.Errorf("expected empty-image error, got %v", err)
	}
}

func TestValidateFavicon_ImageURLAccepted(t *testing.T) {
	cfg := FaviconConfig{Enabled: true, Mode: FaviconModeImage, Image: "https://example.com/icon.png"}
	if err := validateAndResolveFavicon(&cfg, t.TempDir()); err != nil {
		t.Fatalf("URL should be accepted: %v", err)
	}
	if cfg.ResolvedImagePath != "" {
		t.Errorf("URL form should not set ResolvedImagePath, got %q", cfg.ResolvedImagePath)
	}
}

func TestValidateFavicon_RejectsFileURLScheme(t *testing.T) {
	cfg := FaviconConfig{Enabled: true, Mode: FaviconModeImage, Image: "file:///etc/passwd"}
	err := validateAndResolveFavicon(&cfg, t.TempDir())
	// file:// is not http/https-prefixed, so it falls into the local-path
	// branch — Stat will fail. Either way: must error.
	if err == nil {
		t.Errorf("expected error for file:// scheme, got nil")
	}
}

func TestValidateFavicon_LocalPathResolvedRelativeToConfig(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "logo.png")
	if err := os.WriteFile(imgPath, []byte("fake png"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := FaviconConfig{Enabled: true, Mode: FaviconModeImage, Image: "logo.png"}
	if err := validateAndResolveFavicon(&cfg, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ResolvedImagePath != filepath.Clean(imgPath) {
		t.Errorf("ResolvedImagePath = %q, want %q", cfg.ResolvedImagePath, imgPath)
	}
}

func TestValidateFavicon_RejectsBadExtension(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "logo.exe")
	if err := os.WriteFile(imgPath, []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := FaviconConfig{Enabled: true, Mode: FaviconModeImage, Image: "logo.exe"}
	err := validateAndResolveFavicon(&cfg, dir)
	if err == nil || !strings.Contains(err.Error(), "extension") {
		t.Errorf("expected extension error, got %v", err)
	}
}

func TestValidateFavicon_RejectsMissingFile(t *testing.T) {
	cfg := FaviconConfig{Enabled: true, Mode: FaviconModeImage, Image: "does-not-exist.png"}
	err := validateAndResolveFavicon(&cfg, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "not readable") {
		t.Errorf("expected not-readable error, got %v", err)
	}
}

func TestValidateFavicon_RejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "icons.png") // weird but valid extension
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := FaviconConfig{Enabled: true, Mode: FaviconModeImage, Image: "icons.png"}
	err := validateAndResolveFavicon(&cfg, dir)
	if err == nil || !strings.Contains(err.Error(), "directory") {
		t.Errorf("expected directory error, got %v", err)
	}
}
