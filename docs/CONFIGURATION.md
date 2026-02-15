# Configuration Reference

TermBlog is configured via a `config.yaml` file in the working directory.

## Full Configuration Example

```yaml
blog:
  title: "My Terminal Blog"
  description: "A blog you can read in your terminal"
  author: "Your Name"
  base_url: "https://blog.example.com"
  content_dir: "content/posts"
  exit_message: "Thanks for visiting!"
  ascii_header: "header.txt"  # Optional ASCII art file

server:
  ssh_port: 2222
  http_port: 8080
  host_key_path: ".ssh/termblog_host_key"
  rate_limit:
    limit: 10      # Max connections per window
    window: 60     # Window duration in seconds

storage:
  database_path: "termblog.db"

theme: "pipboy"  # Theme name or path to custom theme YAML
```

## Configuration Options

### Blog Section

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `title` | string | "My Terminal Blog" | Blog title shown in header |
| `description` | string | - | Blog description for feeds and meta tags |
| `author` | string | "Anonymous" | Default author for posts |
| `base_url` | string | "http://localhost:8080" | Base URL for feeds and links |
| `content_dir` | string | "content/posts" | Directory containing markdown posts |
| `exit_message` | string | - | Message shown when exiting SSH session |
| `ascii_header` | string | - | Path to ASCII art file for TUI header |

### Server Section

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `ssh_port` | int | 2222 | SSH server port |
| `http_port` | int | 8080 | HTTP server port |
| `host_key_path` | string | ".ssh/termblog_host_key" | Path to SSH host key (auto-generated if missing) |
| `rate_limit.limit` | int | 10 | Maximum connections per time window |
| `rate_limit.window` | int | 60 | Rate limit window in seconds |

### Storage Section

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `database_path` | string | "termblog.db" | Path to SQLite database file |

### Theme

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `theme` | string | "pipboy" | Theme name or path to custom theme file |

## Available Themes

Built-in themes:
- `pipboy` - Retro green terminal (Fallout-inspired)
- `dracula` - Dark theme with vibrant colors
- `nord` - Arctic north-bluish palette
- `monokai` - Classic code editor theme
- `monochrome` - Pure black and white
- `amber` - Classic amber CRT aesthetic
- `matrix` - Green on black digital rain
- `paper` - Light thermal printer aesthetic
- `terminal` - Uses your terminal's native colors and background

## Path Resolution

Relative paths in configuration are resolved relative to the working directory where `termblog` is run.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `TERMBLOG_NO_MOUSE` | Set to "1" to disable mouse support (used by web terminal) |

## File Watcher

When running `termblog serve`, the content directory is automatically watched for changes. Any `.md` file modifications trigger a sync to the database.
