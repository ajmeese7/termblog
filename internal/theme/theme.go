package theme

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Theme represents a color theme for the TUI
type Theme struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description"`
	Colors      ThemeColors `yaml:"colors"`
}

// ThemeColors holds all the color definitions for a theme
type ThemeColors struct {
	Primary    string `yaml:"primary"`
	Secondary  string `yaml:"secondary"`
	Background string `yaml:"background"`
	Text       string `yaml:"text"`
	Muted      string `yaml:"muted"`
	Accent     string `yaml:"accent"`
	Error      string `yaml:"error"`
	Success    string `yaml:"success"`
	Warning    string `yaml:"warning"`
	Border     string `yaml:"border"`
}

// DefaultThemes returns the built-in themes
func DefaultThemes() map[string]*Theme {
	return map[string]*Theme{
		"pipboy": PipBoyTheme(),
		"dracula": DraculaTheme(),
		"nord":    NordTheme(),
		"monokai": MonokaiTheme(),
	}
}

// PipBoyTheme returns the Pip-Boy inspired green theme
func PipBoyTheme() *Theme {
	return &Theme{
		Name:        "Pip-Boy",
		Description: "Retro green terminal aesthetic inspired by Fallout",
		Colors: ThemeColors{
			Primary:    "#00ff00", // Bright green
			Secondary:  "#00cc00", // Slightly darker green
			Background: "#0a0a0a", // Near black
			Text:       "#00ff00", // Bright green
			Muted:      "#006600", // Dark green
			Accent:     "#33ff33", // Light green
			Error:      "#ff3333", // Red
			Success:    "#00ff00", // Green
			Warning:    "#ffcc00", // Amber
			Border:     "#00aa00", // Medium green
		},
	}
}

// DraculaTheme returns the Dracula color theme
func DraculaTheme() *Theme {
	return &Theme{
		Name:        "Dracula",
		Description: "A dark theme with vibrant colors",
		Colors: ThemeColors{
			Primary:    "#bd93f9", // Purple
			Secondary:  "#ff79c6", // Pink
			Background: "#282a36", // Dark gray
			Text:       "#f8f8f2", // Light gray
			Muted:      "#6272a4", // Comment gray
			Accent:     "#50fa7b", // Green
			Error:      "#ff5555", // Red
			Success:    "#50fa7b", // Green
			Warning:    "#ffb86c", // Orange
			Border:     "#44475a", // Selection gray
		},
	}
}

// NordTheme returns the Nord color theme
func NordTheme() *Theme {
	return &Theme{
		Name:        "Nord",
		Description: "An arctic, north-bluish color palette",
		Colors: ThemeColors{
			Primary:    "#88c0d0", // Frost blue
			Secondary:  "#81a1c1", // Darker frost
			Background: "#2e3440", // Polar night
			Text:       "#eceff4", // Snow storm
			Muted:      "#4c566a", // Dark gray
			Accent:     "#a3be8c", // Aurora green
			Error:      "#bf616a", // Aurora red
			Success:    "#a3be8c", // Aurora green
			Warning:    "#ebcb8b", // Aurora yellow
			Border:     "#3b4252", // Darker polar night
		},
	}
}

// MonokaiTheme returns the Monokai color theme
func MonokaiTheme() *Theme {
	return &Theme{
		Name:        "Monokai",
		Description: "The classic Monokai color scheme",
		Colors: ThemeColors{
			Primary:    "#f92672", // Pink
			Secondary:  "#66d9ef", // Cyan
			Background: "#272822", // Dark olive
			Text:       "#f8f8f2", // Light
			Muted:      "#75715e", // Comment brown
			Accent:     "#a6e22e", // Green
			Error:      "#f92672", // Pink
			Success:    "#a6e22e", // Green
			Warning:    "#e6db74", // Yellow
			Border:     "#49483e", // Darker olive
		},
	}
}

// LoadTheme loads a theme from a YAML file
func LoadTheme(path string) (*Theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var theme Theme
	if err := yaml.Unmarshal(data, &theme); err != nil {
		return nil, err
	}

	return &theme, nil
}

// GetTheme returns a theme by name, falling back to built-in themes
func GetTheme(name string, customPath string) *Theme {
	// Try to load custom theme first
	if customPath != "" {
		if theme, err := LoadTheme(customPath); err == nil {
			return theme
		}
	}

	// Fall back to built-in themes
	themes := DefaultThemes()
	if theme, ok := themes[name]; ok {
		return theme
	}

	// Default to Pip-Boy
	return PipBoyTheme()
}
