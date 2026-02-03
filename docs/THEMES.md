# Theme Creation Guide

TermBlog supports custom themes defined in YAML files.

## Theme File Structure

Create a YAML file with the following structure:

```yaml
name: "My Custom Theme"
description: "A description of your theme"
colors:
  primary: "#ffffff"     # Main text/elements color
  secondary: "#cccccc"   # Secondary elements
  background: "#000000"  # Background color
  text: "#ffffff"        # Body text color
  muted: "#666666"       # Muted/disabled text
  accent: "#00ff00"      # Highlighted elements
  error: "#ff0000"       # Error messages
  success: "#00ff00"     # Success messages
  warning: "#ffff00"     # Warning messages
  border: "#333333"      # Border color
```

## Color Roles

| Color | Usage |
|-------|-------|
| `primary` | Main UI elements, selected items, important text |
| `secondary` | Supporting elements, secondary headings |
| `background` | Terminal background color |
| `text` | Standard body text |
| `muted` | Disabled items, hints, metadata |
| `accent` | Highlights, links, interactive elements |
| `error` | Error messages, failed states |
| `success` | Success messages, completed states |
| `warning` | Warning messages, caution states |
| `border` | Lines, separators, borders |

## Using Custom Themes

### Option 1: Config File

Set the `theme` config option to your theme file path:

```yaml
theme: "themes/my-theme.yaml"
```

### Option 2: Themes Directory

Place theme files in a `themes/` directory relative to config:

```
project/
├── config.yaml
├── themes/
│   └── my-theme.yaml
└── content/
```

## Example Themes

### Solarized Dark

```yaml
name: "Solarized Dark"
description: "Ethan Schoonover's Solarized Dark palette"
colors:
  primary: "#93a1a1"
  secondary: "#839496"
  background: "#002b36"
  text: "#839496"
  muted: "#586e75"
  accent: "#2aa198"
  error: "#dc322f"
  success: "#859900"
  warning: "#b58900"
  border: "#073642"
```

### Gruvbox Dark

```yaml
name: "Gruvbox Dark"
description: "Retro groove color scheme"
colors:
  primary: "#ebdbb2"
  secondary: "#d5c4a1"
  background: "#282828"
  text: "#ebdbb2"
  muted: "#928374"
  accent: "#b8bb26"
  error: "#fb4934"
  success: "#b8bb26"
  warning: "#fabd2f"
  border: "#3c3836"
```

### High Contrast

```yaml
name: "High Contrast"
description: "Maximum readability theme"
colors:
  primary: "#ffffff"
  secondary: "#ffffff"
  background: "#000000"
  text: "#ffffff"
  muted: "#888888"
  accent: "#00ffff"
  error: "#ff0000"
  success: "#00ff00"
  warning: "#ffff00"
  border: "#ffffff"
```

## Tips

1. **Contrast** - Ensure sufficient contrast between text and background for readability
2. **Consistency** - Keep primary and text colors similar for visual harmony
3. **Testing** - Test themes with actual content to check readability
4. **Accessibility** - Consider colorblind-friendly palettes

## Theme Persistence

User theme preferences are saved per SSH key fingerprint. When users select a theme with `t`, their choice persists across sessions.
