package styles

import "embed"

//go:embed *.json
var FS embed.FS

// GetStyle returns the JSON style data for a theme
func GetStyle(themeName string) ([]byte, error) {
	return FS.ReadFile(themeName + ".json")
}
