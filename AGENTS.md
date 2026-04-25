# AGENTS.md

Guidelines for AI agents working on this codebase.

## Project Overview

TermBlog is a terminal-based blogging platform written in Go. It serves blog content via SSH (using Wish) and HTTP (a browser-rendered TUI compiled to WASM with [ratzilla](https://crates.io/crates/ratzilla), plus a JSON API and standard SEO routes).

## Architecture

```
cmd/termblog/main.go     → CLI (serve, new, sync, publish, unpublish, delete, list, schedule, version)
internal/app/            → Config loading and app initialization
internal/blog/           → Post model, markdown parsing, feed generation
internal/server/         → SSH server (Wish) and HTTP server (WASM TUI, JSON API, RSS/JSON feeds, sitemap, robots, per-post HTML)
internal/storage/        → SQLite database with migrations
internal/theme/          → Color themes and Lipgloss styles
internal/tui/            → Bubbletea TUI for the SSH transport (list, reader, search, help views)
web/                     → Rust + ratzilla WASM TUI for the HTTP transport (separate from the Bubbletea TUI)
```

## Key Patterns

### Bubbletea TUI
- Each view is a separate model (`ListModel`, `ReaderModel`, `SearchModel`)
- Root `Model` in `app.go` manages view state and routing
- Custom messages (`PostSelectedMsg`, `BackToListMsg`, etc.) for inter-component communication
- Keybindings defined in `keyMap` struct with `key.Binding`

### Database
- SQLite with WAL mode for concurrent access
- Migrations in `internal/storage/migrations/` (SQL files, numbered)
- `PostRepository` handles all post CRUD operations

### Servers
- SSH: Wish middleware chain with Bubbletea handler
- HTTP: stdlib `net/http`; serves the embedded ratzilla WASM TUI at `/`, a JSON API under `/api/`, RSS/JSON feeds, a sitemap, and per-post/per-tag HTML pages. No WebSocket or PTY — the browser TUI runs entirely client-side as WASM.

## Common Tasks

### Adding a new keybinding
1. Add binding to `keyMap` struct in `internal/tui/app.go`
2. Handle in the appropriate view's `Update` method
3. Update help text in `renderHelp()`

### Adding a new view
1. Create new model file in `internal/tui/`
2. Add view constant to `ViewState` enum in `app.go`
3. Add case to `Update` and `View` switch statements
4. Create navigation message type if needed

### Adding a new theme
1. Add theme definition in `internal/theme/theme.go` (or create YAML in `themes/`)
2. Register in `GetTheme()` function

### Database changes
1. Create new migration file: `internal/storage/migrations/00X_name.sql`
2. Migrations run automatically on startup

### Updating the web frontend

**Never hand-edit files under `web/dist/` or `internal/server/wasm_dist/`.** Sources to edit:
- `web/index.html` — HTML shell, inline CSS, theme-prefetch script
- `web/src/*.rs` — Rust/ratzilla TUI code

Then regenerate:
- `make build-wasm` — rebuilds dist + wasm_dist (no Go rebuild)
- `make build-all` — wasm + Go binary

Restart `./termblog serve` after either to pick up changes.

## Build & Test

```bash
go build ./...           # Build all Go packages
go run ./cmd/termblog    # Run directly
./termblog serve         # Start servers after building
make build-wasm          # Rebuild the WASM frontend (regenerates web/dist + internal/server/wasm_dist)
make build-all           # Rebuild WASM + Go binary
```

## Style Guidelines

- Use Bubbletea patterns (Model, Update, View)
- Keep views focused - one file per view model
- Use Lipgloss for all terminal styling
- Vim-style keybindings where applicable
- Messages are the only way to communicate between components

## Documentation

Keep `README.md` updated when making changes that affect:
- CLI commands or flags
- Configuration options
- Keybindings
- Project structure
- Dependencies

## File Locations

- Blog posts: `content/posts/*.md`
- Database: `termblog.db` (SQLite)
- Config: `config.yaml`
- SSH host key: `.ssh/termblog_host_key`

## Tooling Notes

- Use `python3` for Python commands/scripts; do not assume `python` exists.
