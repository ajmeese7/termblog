# TermBlog

A self-hosted, terminal-based blogging platform. Read and write blog posts through SSH or a web-based terminal emulator.

## Features

- **SSH Access** - Connect directly via `ssh localhost -p 2222`
- **Web Terminal** - Browser-based terminal using xterm.js
- **RSS/Atom/JSON Feeds** - Standard feed syndication
- **Markdown Posts** - Write in Markdown with YAML frontmatter
- **Vim-style Navigation** - `j/k`, `ctrl+d/u`, `gg/G`, and more
- **Full-text Search** - Search across titles and tags
- **Theming** - 8 built-in themes (Pip-Boy, Dracula, Nord, Monokai, Monochrome, Amber, Matrix, Paper) plus custom YAML themes
- **Single Binary** - No external runtime dependencies

## Installation

### Download Binary

```bash
# Linux (amd64)
curl -sSL https://github.com/ajmeese7/termblog/releases/latest/download/termblog_*_linux_amd64.tar.gz | tar xz
sudo mv termblog /usr/local/bin/

# macOS (Apple Silicon)
curl -sSL https://github.com/ajmeese7/termblog/releases/latest/download/termblog_*_darwin_arm64.tar.gz | tar xz
sudo mv termblog /usr/local/bin/
```

### Go Install

```bash
go install github.com/ajmeese7/termblog/cmd/termblog@latest
```

### From Source

```bash
git clone https://github.com/ajmeese7/termblog.git
cd termblog
make build

# Run with `./termblog serve` since it's local
```

## Quick Start

```bash
# Start the server
termblog serve

# Access via SSH
ssh localhost -p 2222

# Or open in browser
open http://localhost:8080
```

## Commands

| Command | Description |
|---------|-------------|
| `termblog serve` | Start SSH and HTTP servers |
| `termblog serve --ssh-only` | Start SSH server only |
| `termblog serve --http-only` | Start HTTP server only |
| `termblog new "Post Title"` | Create a new post |
| `termblog sync` | Sync markdown files to database |
| `termblog publish <slug>` | Publish a draft post |
| `termblog unpublish <slug>` | Revert published post to draft |
| `termblog delete <slug>` | Delete a post (use `-r` to remove file) |
| `termblog list` | List all posts with status |
| `termblog schedule <slug> <date>` | Schedule post for future |
| `termblog version` | Show version info |

### SSH Commands (Non-Interactive)

You can pipe data directly via SSH:

```bash
ssh blog.example.com posts              # List all posts
ssh blog.example.com read my-post       # Get raw markdown
ssh blog.example.com rss > feed.xml     # Export RSS feed
ssh blog.example.com search golang      # Search posts
```

## Configuration

Create a `config.yaml` in the project root:

```yaml
blog:
  title: "My Blog"
  description: "A terminal blog"
  author: "Your Name"
  base_url: "https://example.com"

server:
  ssh_port: 2222
  http_port: 8080

storage:
  database: "termblog.db"
  content_dir: "content/posts"

theme:
  name: "dracula"  # pipboy, dracula, nord, monokai
```

## Writing Posts

Posts are Markdown files in `content/posts/` with YAML frontmatter:

```markdown
---
title: "My First Post"
description: "A short description"
author: "Your Name"
date: 2026-02-01
tags: [go, terminal, blog]
draft: false
---

Your content here...
```

## Keybindings

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `ctrl+d` | Half page down |
| `ctrl+u` | Half page up |
| `ctrl+f` / `pgdn` | Page down |
| `ctrl+b` / `pgup` | Page up |
| `g` / `home` | Go to top |
| `G` / `end` | Go to bottom |
| `enter` / `l` | Select/Open post |
| `esc` / `h` | Go back |
| `/` | Search |
| `t` | Theme selector |
| `?` | Toggle help |
| `q` | Quit |

## Testing

### Unit Tests

Go unit tests cover server middleware, rate limiting, SSH commands, TUI rendering, mouse handling, and theme rendering:

```bash
make test
```

### End-to-End Browser Tests

Playwright tests verify the web terminal, theme switching via OSC sequences, localStorage persistence, and page background sync:

```bash
# Install dependencies (first time only)
pushd tests/e2e && npm install && npx playwright install chromium && popd

# Start the server
make build && ./termblog serve &

# Run e2e tests
make test-e2e
```

## Project Structure

```
├── cmd/termblog/       # CLI entry point
├── internal/
│   ├── app/            # Configuration management
│   ├── blog/           # Post parsing, feed generation
│   ├── server/         # SSH and HTTP servers
│   ├── storage/        # SQLite database layer
│   ├── theme/          # Color themes and styling
│   ├── tui/            # Terminal UI (Bubbletea)
│   └── version/        # Build-time version info
├── tests/e2e/          # Playwright browser tests
├── content/posts/      # Markdown blog posts
└── config.yaml         # Configuration
```

## Dependencies

Built with the [Charm](https://charm.sh) ecosystem:
- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Glamour](https://github.com/charmbracelet/glamour) - Markdown rendering
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [Wish](https://github.com/charmbracelet/wish) - SSH server

## Contributing

See [RELEASING.md](./docs/RELEASING.md) for information on the release process and version management.

## License

[BSD 3-Clause](./LICENSE)
