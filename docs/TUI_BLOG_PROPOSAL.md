# TermBlog - A Self-Hosted TUI Blog Platform

## Vision

A self-hosted blogging platform that delivers content through an authentic terminal experience. Readers can SSH directly into your blog or use a browser-native WASM terminal. Think Bearblog meets Terminal.shop meets the Pip-Boy aesthetic.

## Tech Stack

### Core: Go + Charm Ecosystem (SSH/Backend)

| Component | Technology | Rationale |
|-----------|------------|-----------|
| **TUI Framework** | [Bubbletea](https://github.com/charmbracelet/bubbletea) | Battle-tested, beautiful, Go-native |
| **SSH Server** | [Wish](https://github.com/charmbracelet/wish) | Made for exactly this use case |
| **Styling** | [Lipgloss](https://github.com/charmbracelet/lipgloss) | CSS-like styling for terminals |
| **Markdown** | [Glamour](https://github.com/charmbracelet/glamour) | Terminal markdown rendering |
| **Content Storage** | Markdown files + SQLite | Simple, portable, git-friendly |
| **RSS Generation** | [gorilla/feeds](https://github.com/gorilla/feeds) | Standard Go RSS library |

### Web: Rust/WASM + Ratzilla (Browser)

| Component | Technology | Rationale |
|-----------|------------|-----------|
| **TUI Framework** | [Ratzilla](https://github.com/ratzilla-rs/ratzilla) | Official Ratatui WASM project, WebGL2 backend |
| **Widgets** | [Ratatui](https://github.com/ratatui/ratatui) | Re-exported by Ratzilla, rich widget library |
| **Build Tool** | [Trunk](https://trunkrs.dev/) | WASM bundler and dev server |
| **API Client** | web-sys + wasm-bindgen | Browser fetch API for JSON endpoints |
| **Data** | JSON API from Go server | Posts, search, config served as JSON |

### Why This Architecture

The web terminal previously used an xterm.js + WebSocket PTY bridge (browser -> WS -> Go PTY subprocess -> Bubbletea -> PTY -> WS -> browser). Despite optimizations (deflate compression, 60fps batching, WebGL renderer), production latency remained too high for a good web experience — each keystroke round-tripped through the network.

The new architecture keeps Go + Bubbletea for SSH (where it works great) and replaces the web terminal with a Rust/WASM app using Ratzilla. The TUI runs entirely client-side in the browser: zero server-side rendering cost, zero network latency for UI interactions. The Go server just serves the WASM app and a JSON API for blog content.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         TermBlog Server                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐    ┌──────────────────────────────────────┐    │
│  │  SSH Server │    │          HTTP Server                 │    │
│  │   (Wish)    │    │                                      │    │
│  │   :2222     │    │  ┌────────────┐  ┌───────────────┐   │    │
│  │             │    │  │  JSON API  │  │  WASM Assets  │   │    │
│  │  Bubbletea  │    │  │ /api/*     │  │  (embedded)   │   │    │
│  │  TUI        │    │  └────────────┘  └───────────────┘   │    │
│  │  (server-   │    │  ┌────────────┐  ┌───────────────┐   │    │
│  │   side)     │    │  │ Static HTML│  │ Feeds/Sitemap │   │    │
│  │             │    │  │ /archive   │  │ /feed.xml     │   │    │
│  └──────┬──────┘    │  └────────────┘  └───────────────┘   │    │
│         │           └──────────────────────┬───────────────┘    │
│         │                                  │                    │
│         ▼                                  ▼                    │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                    Core Blog Engine                       │  │
│  ├───────────────────────────────────────────────────────────┤  │
│  │  • Post Management (CRUD)                                 │  │
│  │  • RSS/Atom/JSON Feed Generator                           │  │
│  │  • Full-Text Search (FTS5)                                │  │
│  │  • View Analytics                                         │  │
│  └───────────────────────────────────────────────────────────┘  │
│                              │                                  │
│                              ▼                                  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                    Storage Layer                          │  │
│  ├───────────────────────────────────────────────────────────┤  │
│  │  • SQLite (metadata, FTS5, views, preferences)            │  │
│  │  • Filesystem (markdown posts)                            │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

                    ┌─────────────────────────┐
                    │      Browser Client     │
                    ├─────────────────────────┤
                    │  Ratzilla/Ratatui WASM  │
                    │  (WebGL2 canvas)        │
                    │                         │
                    │  • Client-side TUI      │
                    │  • fetch() → JSON API   │
                    │  • localStorage themes  │
                    │  • Zero latency UI      │
                    └─────────────────────────┘
```

## User Experiences

### 1. Reader via SSH
```
$ ssh blog.yoursite.com

╭─────────────────────────────────────────────────╮
│            AARON'S TERMINAL BLOG                │
│         ══════════════════════════              │
│                                                 │
│  [↑/↓] Navigate   [Enter] Read   [/] Search     │
│  [r] RSS Info     [t] Theme      [q] Quit       │
╰─────────────────────────────────────────────────╯

  RECENT POSTS
  ────────────
  > Building a TUI Blog Platform        2026-01-28
    Why I Left Big Tech                 2026-01-15
    Rust vs Go for CLI Tools            2026-01-03
    The Beauty of Monochrome Design     2025-12-20

  [Page 1/5]  ─────────────────────────  20 posts
```

### 2. Reader via Web (WASM)
Browser loads a page with the Ratzilla WASM app. The TUI renders directly on a WebGL2 canvas — same visual appearance as SSH, but running entirely client-side. Blog data is fetched from the JSON API. Theme changes are saved to localStorage and applied instantly.

### 3. Reader via Static HTML (Fallback)
Clean, minimal HTML with terminal-inspired CSS at `/archive`, `/posts/:slug`, `/tags/:tag`. Full SEO, works without JS.

### 4. Author/Admin Experience
```
$ ssh admin@blog.yoursite.com

╭─────────────────────────────────────────────────╮
│            TERMBLOG ADMIN CONSOLE               │
╰─────────────────────────────────────────────────╯

  [n] New Post     [e] Edit Post    [d] Delete Post
  [p] Publish      [u] Unpublish    [t] Themes
  [s] Settings     [a] Analytics    [q] Quit

  DRAFTS (3)
  ──────────
  > My New Post Idea                   (modified 2h ago)
    Another Draft                      (modified 1d ago)

  SCHEDULED (1)
  ─────────────
    Upcoming Post                      publishes 2026-02-05
```

---

## Detailed Implementation TODO

### Phase 1: Core Infrastructure (Foundation) ✅ COMPLETE

#### 1.1 Project Setup
- [x] Go module, directory structure, core dependencies
- [x] Configuration system (YAML-based)
- [x] Basic logging

#### 1.2 Storage Layer
- [x] SQLite schema (posts, settings, user_preferences, views)
- [x] Post repository (CRUD operations)
- [x] Filesystem watcher for hot-reload
- [x] Markdown parser with frontmatter support
- [x] Full-text search (FTS5)

#### 1.3 Blog Engine Core
- [x] Post listing with pagination
- [x] Post retrieval by slug
- [x] Tag-based filtering
- [x] RSS/Atom/JSON feed generation
- [x] Sitemap generation

### Phase 2: TUI Reader Interface ✅ COMPLETE

#### 2.1 Bubbletea Models
- [x] Main menu, post list (vim navigation), post reader (Glamour), search, theme selector

#### 2.2 Terminal Themes System
- [x] 9 built-in themes: Pip-Boy, Dracula, Nord, Monokai, Monochrome, Amber, Matrix, Paper, Terminal
- [x] Theme persistence (per SSH fingerprint)
- [x] ASCII art header support

### Phase 3: SSH Server ✅ COMPLETE

- [x] Wish integration with host key management
- [x] Rate limiting, connection logging, MOTD
- [x] Non-interactive SSH commands (posts, read, rss, search, help)
- [x] Public key auth for fingerprint tracking

### Phase 4: Web Interface ✅ COMPLETE

#### 4.1 JSON API
- [x] `GET /api/posts?page=1&per_page=10` — paginated published posts
- [x] `GET /api/posts/{slug}` — full post with raw markdown
- [x] `GET /api/search?q=query&limit=20` — FTS5 search
- [x] `GET /api/tags` — all tags with counts
- [x] `GET /api/tags/{tag}` — posts filtered by tag
- [x] `GET /api/config` — blog title, description, themes, ASCII header
- [x] `POST /api/views/{slug}` — record view (hashed IP)
- [x] CORS headers for local dev

#### 4.2 Ratzilla WASM App (`web/`)
- [x] Ratzilla + Ratatui scaffold with WebGL2 backend
- [x] API client (web-sys fetch)
- [x] Post list view (cursor, pagination, vim keys)
- [x] Post reader view (markdown rendering, scrolling)
- [x] Search view (text input, debounced API search)
- [x] Theme selector (9 themes, localStorage persistence, live preview)
- [x] Help overlay (keybinding reference)
- [x] Loading states and error handling

#### 4.3 Static HTML Fallback
- [x] Archive, post pages, tag pages with terminal-inspired CSS
- [x] Zero JavaScript requirement (for static pages)
- [x] OpenGraph/Twitter meta tags

#### 4.4 HTTP Server
- [x] Route handling: `/` (WASM app), `/api/*`, `/archive`, `/posts/:slug`, `/tags/:tag`, feeds, health
- [x] Embedded WASM assets (`go:embed wasm_dist`)
- [x] Gzip compression, cache headers

### Phase 5: Admin Interface ✅ COMPLETE

- [x] SSH key fingerprint authentication
- [x] Post management via TUI (create, edit, delete, publish/unpublish)
- [x] CLI management tool (new, publish, unpublish, delete, list, schedule, sync, serve, version)

### Phase 6: Advanced Features 🔶 PARTIAL

- [x] Code syntax highlighting (Glamour/Chroma)
- [x] Reading time estimates
- [x] Table of contents generation
- [x] View counter/analytics
- [ ] Image support (ASCII art conversion)
- [ ] Post series/collections
- [ ] Import tool for Hugo/Jekyll migration

### Phase 7: Build & Deployment ✅ COMPLETE

- [x] Makefile with `build-wasm`, `build-all`, `clean-all` targets
- [x] Multi-stage Dockerfile (Rust WASM build → Go build → Alpine)
- [x] GitHub Actions release workflow (Rust toolchain + WASM build)
- [x] Docker Compose with Caddy example
- [x] Health check endpoint

### Phase 8: Documentation & Polish 🔶 PARTIAL

- [x] README with quick start and Rust build instructions
- [x] Configuration reference, theme creation guide, release process
- [ ] Migration guides (from Hugo, Jekyll)
- [ ] Security audit
- [ ] Cross-terminal testing

---

## Current Project Status

### Completed Features
- Full TUI reader with 9 built-in themes (SSH via Bubbletea)
- WASM web terminal via Ratzilla/Ratatui (client-side, zero latency)
- JSON API for blog content
- SSH server with rate limiting, MOTD, non-interactive commands
- Static HTML pages (archive, posts, tags) with SEO tags
- RSS/Atom/JSON feeds and sitemap
- CLI tools: new, publish, unpublish, delete, list, schedule, sync, serve
- Docker + Docker Compose support
- GitHub Actions release workflow
- File watcher for auto-sync
- Theme persistence (SSH: fingerprint, Web: localStorage)
- View counter/analytics

### Remaining Work

**Medium Priority:**
- [ ] Tag filtering in TUI (only via web /tags/:tag currently)
- [ ] Import tool for Hugo/Jekyll migration

**Low Priority/Nice-to-have:**
- [ ] Image support (ASCII art conversion)
- [ ] Post series/collections
- [ ] Webmention/comments
- [ ] Systemd service file
- [ ] Security audit
- [ ] Cross-terminal testing

---

## Estimated Complexity

| Phase | Complexity | Status |
|-------|------------|--------|
| Phase 1: Core Infrastructure | Medium | ✅ Complete |
| Phase 2: TUI Reader | High | ✅ Complete |
| Phase 3: SSH Server | Low | ✅ Complete |
| Phase 4: Web Interface (WASM + API) | High | ✅ Complete |
| Phase 5: Admin Interface | Medium | ✅ Complete |
| Phase 6: Advanced Features | Variable | 🔶 Partial |
| Phase 7: Build & Deployment | Medium | ✅ Complete |
| Phase 8: Documentation & Polish | Low | 🔶 Partial |
