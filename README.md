# TermBlog

A self-hosted, terminal-based blogging platform. Read and write blog posts through SSH or a web-based terminal emulator.

## Features

- **SSH Access** - Connect directly via `ssh localhost -p 2222`
- **Web Terminal** - Browser-based terminal using xterm.js
- **RSS/Atom/JSON Feeds** - Standard feed syndication
- **Markdown Posts** - Write in Markdown with YAML frontmatter
- **Vim-style Navigation** - `j/k`, `ctrl+d/u`, `gg/G`, and more
- **Full-text Search** - Search across titles and tags
- **Theming** - Built-in themes (Pip-Boy, Dracula, Nord, Monokai) plus custom YAML themes
- **Single Binary** - No external runtime dependencies

## Quick Start

```bash
# Build
go build -o termblog ./cmd/termblog

# Start the server
./termblog serve

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
| `?` | Toggle help |
| `q` | Quit |

## Project Structure

```
├── cmd/termblog/       # CLI entry point
├── internal/
│   ├── app/            # Configuration management
│   ├── blog/           # Post parsing, feed generation
│   ├── server/         # SSH and HTTP servers
│   ├── storage/        # SQLite database layer
│   ├── theme/          # Color themes and styling
│   └── tui/            # Terminal UI (Bubbletea)
├── content/posts/      # Markdown blog posts
└── config.yaml         # Configuration
```

## Dependencies

Built with the [Charm](https://charm.sh) ecosystem:
- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Glamour](https://github.com/charmbracelet/glamour) - Markdown rendering
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [Wish](https://github.com/charmbracelet/wish) - SSH server

## License

[BSD 3-Clause](./LICENSE)
