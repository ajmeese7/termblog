# TermBlog

A self-hosted, terminal-based blogging platform. Read and write blog posts through SSH or a browser-native WASM terminal.

## Features

- **SSH Access** - Connect directly via `ssh localhost -p 2222`
- **WASM Web Terminal** - Client-side TUI via Ratzilla/Ratatui
- **JSON API** - RESTful API for blog content (`/api/posts`, `/api/search`, etc.)
- **RSS/Atom/JSON Feeds** - Standard feed syndication
- **Markdown Posts** - Write in Markdown with YAML frontmatter
- **Vim-style Navigation** - `j/k`, `ctrl+d/u`, `gg/G`, and more
- **Full-text Search** - SQLite FTS5 search across titles and content
- **Theming** - 9 built-in themes (Pip-Boy, Dracula, Nord, Monokai, Monochrome, Amber, Matrix, Paper, Terminal) plus custom YAML themes
- **Theme-aware Favicon** - Browser tab icon recolors live with the active theme; configurable as a letter, an emoji, or a custom image (local file or URL)
- **Single Binary** - Go server with embedded WASM assets, no external runtime dependencies

## Architecture

```
┌──────────────────────────────────┐     ┌─────────────────────┐
│         Go Server                │     │   Browser (WASM)    │
│                                  │     │                     │
│  SSH Server (Wish, :2222)        │     │  Ratzilla/Ratatui   │
│    └─ Bubbletea TUI              │     │  DOM backend        │
│                                  │     │    │                │
│  HTTP Server (:8080)             │◄────│    └─ fetch /api/*  │
│    ├─ JSON API  (/api/*)         │     │                     │
│    ├─ WASM App  (/)              │────►│  (served as static) │
│    ├─ Static HTML (/archive)     │     └─────────────────────┘
│    └─ Feeds (/feed.xml)          │
│                                  │
│  SQLite + Markdown files         │
└──────────────────────────────────┘
```

SSH readers get the Bubbletea TUI rendered server-side. Web readers get a Rust/WASM app (built with Ratzilla) that runs entirely client-side and fetches blog data from the JSON API. Static HTML fallback pages provide SEO and accessibility.

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

Building from source requires both Go and Rust toolchains:

```bash
git clone https://github.com/ajmeese7/termblog.git
cd termblog

# Build everything (WASM + Go binary)
make build-all

# Or build components individually:
make build-wasm   # Build Rust/WASM app (requires Rust + Trunk)
make build        # Build Go binary (embeds WASM assets)
```

#### Rust/WASM Prerequisites

```bash
# Install Rust (if not already installed)
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# Add WASM target
rustup target add wasm32-unknown-unknown

# Install Trunk (WASM bundler)
cargo install trunk
```

## Quick Start

```bash
# Start the server
termblog serve

# or if you built from source:
# ./termblog serve

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
ssh -p 2222 blog.example.com posts              # List all posts
ssh -p 2222 blog.example.com read my-post       # Get raw markdown
ssh -p 2222 blog.example.com rss > feed.xml     # Export RSS feed
ssh -p 2222 blog.example.com search golang      # Search posts
```

### JSON API

The HTTP server exposes a JSON API for the WASM app (and any other clients):

```bash
curl localhost:8080/api/posts                    # Paginated post list
curl localhost:8080/api/posts/my-post            # Full post with markdown
curl localhost:8080/api/search?q=golang          # Full-text search
curl localhost:8080/api/tags                     # All tags with counts
curl localhost:8080/api/config                   # Blog configuration
```

### Production SSH with Cloudflare

If your domain uses Cloudflare orange-cloud proxying, raw SSH on `2222` will time out.

For standard OpenSSH access, use a dedicated SSH hostname that is DNS-only (gray cloud), then tell visitors to connect with:

```bash
ssh -p 2222 ssh.example.com
```

Keep `termblog.com` proxied for HTTPS, and use `ssh.example.com` for SSH.

## Configuration

Create a `config.yaml` in the project root:

```sh
cp example.config.yaml config.yaml
```

See [Configuration Reference](./docs/CONFIGURATION.md) for all options.

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
canonical_url: "https://example.com/original-post"
---

Your content here...
```

### Frontmatter Fields

| Field | Required | Description |
|-------|----------|-------------|
| `title` | yes | Post title |
| `description` | no | Short description for SEO and feeds |
| `author` | no | Post author (defaults to blog author) |
| `date` | yes | Creation date (`YYYY-MM-DD`) |
| `tags` | no | List of tags |
| `draft` | no | Set `true` to hide from readers (default `false`) |
| `published_at` | no | Explicit publish date (defaults to `date`) |
| `canonical_url` | no | Original URL for content migrated from another platform. Sets `<link rel="canonical">` to avoid duplicate content SEO penalties |

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
| `y` | Copy post link to clipboard (web) |
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

Playwright tests verify the web terminal, theme switching, and localStorage persistence:

```bash
# Install dependencies (first time only)
pushd tests/e2e && npm install && npx playwright install chromium && popd

# Start the server
make build-all && ./termblog serve &

# Run e2e tests
make test-e2e
```

## Development

### WASM Frontend (Hot Reload)

The Rust/WASM frontend can be developed with hot reloading using [Trunk](https://trunkrs.dev/). The `Trunk.toml` proxies `/api/*` requests to the Go server, so both can run simultaneously:

```bash
# Terminal 1: Start the Go server (port 8080)
make build && ./termblog serve

# Terminal 2: Start Trunk dev server with hot reload (port 9090)
cd web && trunk serve --port 9090
```

Open `http://localhost:9090` in your browser. Trunk watches `web/src/` and `web/index.html` for changes, automatically rebuilding and reloading the browser. API requests are proxied to the Go server on port 8080.

When ready to test the full embedded build:

```bash
make build-all    # Build WASM, then embed into Go binary
./termblog serve  # Serves everything from a single binary
```

## Project Structure

```
├── cmd/termblog/       # CLI entry point
├── internal/
│   ├── app/            # Configuration management
│   ├── blog/           # Post parsing, feed generation
│   ├── server/         # SSH server, HTTP server, JSON API
│   ├── storage/        # SQLite database layer
│   ├── theme/          # Color themes and styling
│   ├── tui/            # Terminal UI (Bubbletea, SSH)
│   └── version/        # Build-time version info
├── web/                # Rust/WASM app (Ratzilla/Ratatui)
│   ├── Cargo.toml
│   ├── Trunk.toml
│   ├── index.html
│   └── src/            # Rust source (views, API client, themes)
├── tests/e2e/          # Playwright browser tests
├── content/posts/      # Markdown blog posts
└── config.yaml         # Configuration
```

## Dependencies

### Go (SSH + Backend)
- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Glamour](https://github.com/charmbracelet/glamour) - Markdown rendering
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [Wish](https://github.com/charmbracelet/wish) - SSH server

### Rust (Web WASM)
- [Ratzilla](https://github.com/ratzilla-rs/ratzilla) - Ratatui WASM backend
- [Ratatui](https://github.com/ratatui/ratatui) - TUI widget library
- [Trunk](https://trunkrs.dev/) - WASM build tool

## Documentation

- [Deployment Guide](./docs/DEPLOYMENT.md)
- [Configuration Reference](./docs/CONFIGURATION.md)
- [Theme Creation Guide](./docs/THEMES.md)
- [Release Process](./docs/RELEASING.md)

## License

[BSD 3-Clause](./LICENSE)
