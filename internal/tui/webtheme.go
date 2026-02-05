package tui

import (
	"fmt"
	"os"
)

// isWebMode returns true if running in web PTY mode
func isWebMode() bool {
	return os.Getenv("TERMBLOG_NO_MOUSE") == "1"
}

// emitWebThemeChange sends an OSC escape sequence to notify the web terminal of a theme change.
// Format: OSC 7777 ; theme=<themename> BEL
// This is only emitted in web mode (when TERMBLOG_NO_MOUSE=1).
func emitWebThemeChange(themeName string) string {
	if !isWebMode() {
		return ""
	}
	// OSC 7777 (custom code) with theme name, terminated by BEL (\x07)
	return fmt.Sprintf("\x1b]7777;theme=%s\x07", themeName)
}
