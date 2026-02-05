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
		"pipboy":     PipBoyTheme(),
		"dracula":    DraculaTheme(),
		"nord":       NordTheme(),
		"monokai":    MonokaiTheme(),
		"monochrome": MonochromeTheme(),
		"amber":      AmberTheme(),
		"matrix":     MatrixTheme(),
		"paper":      PaperTheme(),
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

// MonochromeTheme returns a pure black and white theme
func MonochromeTheme() *Theme {
	return &Theme{
		Name:        "Monochrome",
		Description: "Pure black and white minimalist theme",
		Colors: ThemeColors{
			Primary:    "#ffffff", // White
			Secondary:  "#cccccc", // Light gray
			Background: "#000000", // Black
			Text:       "#ffffff", // White
			Muted:      "#666666", // Gray
			Accent:     "#ffffff", // White
			Error:      "#ff0000", // Red (only color)
			Success:    "#ffffff", // White
			Warning:    "#ffffff", // White
			Border:     "#444444", // Dark gray
		},
	}
}

// AmberTheme returns an amber CRT terminal aesthetic
func AmberTheme() *Theme {
	return &Theme{
		Name:        "Amber",
		Description: "Classic amber CRT terminal aesthetic",
		Colors: ThemeColors{
			Primary:    "#ffb000", // Amber
			Secondary:  "#ff8c00", // Dark amber
			Background: "#0d0800", // Very dark brown/black
			Text:       "#ffb000", // Amber
			Muted:      "#805800", // Dark amber
			Accent:     "#ffc740", // Light amber
			Error:      "#ff4500", // Orange red
			Success:    "#ffb000", // Amber
			Warning:    "#ffd700", // Gold
			Border:     "#996600", // Medium amber
		},
	}
}

// MatrixTheme returns a green-on-black digital rain aesthetic
func MatrixTheme() *Theme {
	return &Theme{
		Name:        "Matrix",
		Description: "Green on black digital rain aesthetic",
		Colors: ThemeColors{
			Primary:    "#00ff41", // Matrix green
			Secondary:  "#008f11", // Dark matrix green
			Background: "#0d0208", // Near black with slight green
			Text:       "#00ff41", // Matrix green
			Muted:      "#006b00", // Dark green
			Accent:     "#39ff14", // Neon green
			Error:      "#ff0000", // Red (system error)
			Success:    "#00ff41", // Matrix green
			Warning:    "#00ff41", // Matrix green
			Border:     "#006400", // Dark green
		},
	}
}

// PaperTheme returns a light theme with thermal printer aesthetic
func PaperTheme() *Theme {
	return &Theme{
		Name:        "Paper",
		Description: "Light theme with thermal printer aesthetic",
		Colors: ThemeColors{
			Primary:    "#1a1a1a", // Near black text
			Secondary:  "#4a4a4a", // Dark gray
			Background: "#f5f5dc", // Beige/cream paper
			Text:       "#1a1a1a", // Near black
			Muted:      "#8b8b7a", // Muted olive gray
			Accent:     "#2e2e2e", // Slightly lighter black
			Error:      "#8b0000", // Dark red
			Success:    "#1a1a1a", // Near black
			Warning:    "#654321", // Dark brown
			Border:     "#c0c0a8", // Light olive
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
