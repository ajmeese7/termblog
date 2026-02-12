# AGENTS.md

Guidelines for AI agents working on this codebase.

## Project Overview

TermBlog is a terminal-based blogging platform written in Go. It serves blog content via SSH (using Wish) and HTTP (with a web-based terminal via xterm.js).

## Architecture

```
cmd/termblog/main.go     → CLI commands (serve, new, sync, pty)
internal/app/            → Config loading and app initialization
internal/blog/           → Post model, markdown parsing, feed generation
internal/server/         → SSH server (Wish) and HTTP server (WebSocket + PTY)
internal/storage/        → SQLite database with migrations
internal/theme/          → Color themes and Lipgloss styles
internal/tui/            → Bubbletea TUI (list, reader, search, help views)
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
- HTTP: Standard library + Gorilla WebSocket, spawns PTY process

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

## Build & Test

```bash
go build ./...           # Build all packages
go run ./cmd/termblog    # Run directly
./termblog serve         # Start servers after building
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
