# TermBlog - A Self-Hosted TUI Blog Platform

## Vision

A self-hosted blogging platform that delivers content through an authentic terminal experience. Readers can SSH directly into your blog or use a web-based terminal emulator. Think Bearblog meets Terminal.shop meets the Pip-Boy aesthetic.

## Why This Approach Over Alternatives

### Considered Alternatives

| Approach | Pros | Cons |
|----------|------|------|
| **Static site with CSS terminal theme** | Simple, fast, SEO-friendly | Feels fake, no real interactivity |
| **xterm.js only (web terminal)** | Works in browser, no SSH needed | Still feels like a gimmick |
| **SSH-only (like terminal.shop)** | Authentic experience | Limits audience, no SEO |
| **Hybrid: SSH + Web Terminal + Static** | Best of all worlds | More complexity |

**Recommendation: Hybrid approach** - This gives you:
- SSH access for terminal purists (`ssh blog.yoursite.com`)
- Web-based terminal for casual visitors
- Static HTML fallback for SEO/RSS/accessibility

## Recommended Tech Stack

### Core: Go + Charm Ecosystem

| Component | Technology | Rationale |
|-----------|------------|-----------|
| **TUI Framework** | [Bubbletea](https://github.com/charmbracelet/bubbletea) | Battle-tested, beautiful, Go-native |
| **SSH Server** | [Wish](https://github.com/charmbracelet/wish) | Made for exactly this use case |
| **Styling** | [Lipgloss](https://github.com/charmbracelet/lipgloss) | CSS-like styling for terminals |
| **Markdown** | [Glamour](https://github.com/charmbracelet/glamour) | Terminal markdown rendering |
| **Web Terminal** | [xterm.js](https://xtermjs.org/) | Industry standard web terminal |
| **Web Framework** | Minimal Go HTTP or [Echo](https://echo.labstack.com/) | Lightweight, same language |
| **Content Storage** | Markdown files + SQLite | Simple, portable, git-friendly |
| **RSS Generation** | [gorilla/feeds](https://github.com/gorilla/feeds) | Standard Go RSS library |

### Why Go + Charm Over Alternatives

- **Rust + Ratatui**: Great library, but Charm ecosystem has SSH (Wish) built-in; Rust lacks equivalent
- **Python + Textual**: Good for TUI, but SSH story is weaker and performance matters for hosting
- **Node.js**: No mature TUI library comparable to Bubbletea

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         TermBlog Server                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐ │
│  │  SSH Server │    │ HTTP Server │    │   Admin Interface   │ │
│  │   (Wish)    │    │   (Echo)    │    │   (Web or TUI)      │ │
│  │   :22       │    │   :443      │    │   :8080             │ │
│  └──────┬──────┘    └──────┬──────┘    └──────────┬──────────┘ │
│         │                  │                      │            │
│         ▼                  ▼                      ▼            │
│  ┌─────────────────────────────────────────────────────────────┤
│  │                    Core Blog Engine                         │
│  ├─────────────────────────────────────────────────────────────┤
│  │  • Post Management (CRUD)                                   │
│  │  • Theme Engine (terminal color schemes)                    │
│  │  • RSS Feed Generator                                       │
│  │  • Search Index                                             │
│  └─────────────────────────────────────────────────────────────┤
│                              │                                 │
│                              ▼                                 │
│  ┌─────────────────────────────────────────────────────────────┤
│  │                    Storage Layer                            │
│  ├─────────────────────────────────────────────────────────────┤
│  │  • SQLite (metadata, themes, settings)                      │
│  │  • Filesystem (markdown posts, assets)                      │
│  └─────────────────────────────────────────────────────────────┘
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## User Experiences

### 1. Reader via SSH
```
$ ssh blog.yoursite.com

╭─────────────────────────────────────────────────╮
│            AARON'S TERMINAL BLOG                │
│         ══════════════════════════              │
│                                                 │
│  [↑/↓] Navigate   [Enter] Read   [/] Search    │
│  [r] RSS Info     [t] Theme      [q] Quit      │
╰─────────────────────────────────────────────────╯

  RECENT POSTS
  ────────────
  > Building a TUI Blog Platform        2026-01-28
    Why I Left Big Tech                 2026-01-15
    Rust vs Go for CLI Tools            2026-01-03
    The Beauty of Monochrome Design     2025-12-20

  [Page 1/5]  ─────────────────────────  20 posts
```

### 2. Reader via Web Terminal
Browser loads a black page with xterm.js that connects via WebSocket to the same TUI, giving identical experience.

### 3. Reader via Static HTML (Fallback)
Clean, minimal HTML with optional terminal-inspired CSS. Full SEO, works without JS.

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

Alternatively, a web-based admin panel at `/admin` with a simple markdown editor.

---

## Detailed Implementation TODO

### Phase 1: Core Infrastructure (Foundation) ✅ COMPLETE

#### 1.1 Project Setup
- [x] Initialize Go module (`go mod init github.com/ajmeese7/termblog`)
- [x] Set up directory structure:
  ```
  termblog/
  ├── cmd/
  │   └── termblog/
  │       └── main.go
  ├── internal/
  │   ├── app/            # Configuration management
  │   ├── blog/           # Core blog logic
  │   ├── server/         # SSH and HTTP servers
  │   ├── tui/            # Bubbletea models
  │   ├── storage/        # SQLite + filesystem
  │   ├── theme/          # Terminal themes
  │   └── version/        # Build-time version info
  ├── content/
  │   └── posts/          # Markdown files
  └── config.yaml
  ```
- [x] Add core dependencies:
  - `github.com/charmbracelet/bubbletea`
  - `github.com/charmbracelet/wish`
  - `github.com/charmbracelet/lipgloss`
  - `github.com/charmbracelet/glamour`
  - `github.com/mattn/go-sqlite3`
  - `gopkg.in/yaml.v3`
  - `github.com/fsnotify/fsnotify` (file watching)
  - `github.com/gorilla/feeds` (RSS/Atom/JSON feeds)
  - `github.com/gorilla/websocket` (web terminal)
  - `github.com/spf13/cobra` (CLI)
- [x] Create configuration system (YAML-based)
- [x] Set up basic logging

#### 1.2 Storage Layer
- [x] Design SQLite schema:
  ```sql
  -- Posts metadata (content in filesystem)
  CREATE TABLE posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    filepath TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft', 'published', 'scheduled')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    published_at DATETIME,
    tags TEXT DEFAULT '[]'
  );

  -- Settings
  CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
  );

  -- User preferences (theme by SSH fingerprint)
  CREATE TABLE user_preferences (
    fingerprint TEXT PRIMARY KEY,
    theme TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
  );
  ```
- [x] Implement post repository (CRUD operations)
- [x] Implement filesystem watcher for hot-reload of markdown files
- [x] Build markdown parser with frontmatter support

#### 1.3 Blog Engine Core
- [x] Post listing with pagination
- [x] Post retrieval by slug
- [x] Tag-based filtering (via HTTP routes /tags/:tag)
- [x] Search (LIKE-based on titles and tags)
- [ ] Full-text search (SQLite FTS5) - *using LIKE instead*
- [x] RSS/Atom/JSON feed generation
- [x] Sitemap generation

### Phase 2: TUI Reader Interface ✅ COMPLETE

#### 2.1 Bubbletea Models
- [x] **Main Menu Model**: Navigation hub
  - Recent posts list
  - Search prompt
  - Theme selector
  - Help overlay
- [x] **Post List Model**: Paginated, filterable post listing
  - Vim-style navigation (j/k/gg/G, ctrl+d/u, ctrl+f/b)
  - Mouse scrolling (SSH only)
- [ ] Tag filtering in TUI (only via web /tags/:tag)
- [ ] Sort options (date, title) - *only date sorting*
- [x] **Post Reader Model**: Full post display
  - Glamour markdown rendering
  - Scrolling (mouse + keyboard)
  - "Back to list" navigation
- [ ] Link extraction and display
- [x] **Search Model**: Interactive search
  - Real-time filtering
  - Search in titles + tags
- [ ] Highlight matches
- [x] **Theme Selector Model**: Live theme preview
  - Cycle through installed themes
  - Instant preview

#### 2.2 Terminal Themes System
- [x] Design theme specification format (Go structs with YAML loading support)
- [x] Implement built-in themes:
  - [x] **Monochrome**: Pure black and white
  - [x] **Pip-Boy**: Green phosphor CRT aesthetic
  - [x] **Amber**: Amber monochrome terminal
  - [x] **Matrix**: Green on black, "digital rain" feel
  - [x] **Paper**: Light theme, like thermal printer output
  - [x] **Dracula**: Dark theme with vibrant colors
  - [x] **Nord**: Arctic, north-bluish palette
  - [x] **Monokai**: Classic Monokai scheme
- [x] Theme persistence (remember user choice via SSH key fingerprint)
- [x] Custom theme loading from YAML files
- [ ] Theme hot-reload for development (themes are Go code)

#### 2.3 Visual Polish
- [x] ASCII art header/logo support (config option)
- [ ] Animated transitions (subtle, optional)
- [x] Loading states ("Loading..." message)
- [x] Box drawing characters for borders (Lipgloss)
- [x] Status bar with current location/hints

### Phase 3: SSH Server ✅ COMPLETE

#### 3.1 Wish Integration
- [x] Basic SSH server setup with Wish
- [x] Host key generation and management
- [x] Exit message display after TUI closes
- [x] MOTD support (config option)
- [x] Graceful connection handling
- [x] Rate limiting (configurable limit/window)
- [x] Connection logging

#### 3.2 Authentication (Optional)
- [x] Public key authentication (for fingerprint tracking)
- [x] Anonymous read access (default)
- [ ] `~/.ssh/authorized_keys` integration for admin

#### 3.3 SSH Commands (Non-TUI Mode)
- [x] `ssh blog.yoursite.com posts` - List posts as plain text
- [x] `ssh blog.yoursite.com read <slug>` - Output post to stdout
- [x] `ssh blog.yoursite.com read <slug> --rendered` - Output plain text
- [x] `ssh blog.yoursite.com rss` - Output RSS feed
- [x] `ssh blog.yoursite.com search <query>` - Search posts
- [x] `ssh blog.yoursite.com help` - Show available commands

### Phase 4: Web Interface ✅ COMPLETE

#### 4.1 Web Terminal (xterm.js)
- [x] Minimal HTML page that loads xterm.js
- [x] WebSocket proxy to PTY backend
- [x] Mobile-responsive terminal sizing
- [x] Copy/paste support (Ctrl+C copies selection)
- [x] Connection status indicator
- [x] Reconnection logic with exponential backoff
- [x] Theme sync via OSC 7777 sequences
- [x] Theme persistence in localStorage
- [ ] Touch support for mobile (basic xterm.js only)

#### 4.2 Static HTML Fallback
- [x] Minimal HTML templates for:
  - [x] Archive/post list (/archive)
  - [x] Individual post pages (/posts/:slug)
  - [x] Tag pages (/tags/:tag)
  - [ ] RSS feed page (with subscription instructions)
- [x] Terminal-inspired CSS theme (uses config theme colors)
- [x] Zero JavaScript requirement (for static pages)
- [x] Proper semantic HTML for accessibility
- [x] OpenGraph/Twitter meta tags

#### 4.3 HTTP Server
- [x] Route handling:
  - `GET /` - Web terminal
  - `GET /archive` - Post list (static HTML)
  - `GET /posts/:slug` - Post page
  - `GET /tags/:tag` - Tag listing
  - `GET /feed.xml` - RSS feed
  - `GET /feed.json` - JSON feed
  - `GET /sitemap.xml` - Sitemap
  - `GET /robots.txt` - Robots file
  - `GET /ws` - WebSocket for terminal
  - `GET /health` - Health check endpoint
- [ ] Content negotiation (HTML vs JSON API)
- [x] Static file serving (embedded)
- [x] Gzip compression middleware
- [x] Cache headers

### Phase 5: Admin Interface 🔶 PARTIAL

#### 5.1 Admin TUI (via SSH)
- [ ] Authenticate via SSH key
- [ ] Post management via TUI:
  - [ ] Create new post (opens $EDITOR or built-in)
  - [ ] Edit existing post
  - [ ] Delete post (with confirmation)
  - [ ] Publish/unpublish toggle
  - [ ] Schedule post for future
- [ ] Theme management via TUI
- [ ] Settings via TUI
- [ ] Simple analytics (view counts, popular posts)

#### 5.2 Admin Web UI (Alternative)
- [ ] Simple login page (password or SSO)
- [ ] Markdown editor with live preview
- [ ] File upload for images
- [ ] Draft management
- [ ] Settings panel

#### 5.3 CLI Management Tool
- [x] `termblog new "Post Title"` - Create new post file
- [x] `termblog publish <slug>` - Publish a draft
- [x] `termblog unpublish <slug>` - Revert to draft
- [x] `termblog delete <slug>` - Delete post (optional -r to remove file)
- [x] `termblog list` - List all posts with status
- [x] `termblog schedule <slug> <datetime>` - Schedule post for future
- [x] `termblog sync` - Sync markdown files to database
- [x] `termblog serve` - Start server (--ssh-only, --http-only flags)
- [x] `termblog version` - Show version info (--full for commit/date)
- [ ] `termblog import <hugo|jekyll|bearblog>` - Import from other platforms

### Phase 6: Advanced Features 🔶 PARTIAL

#### 6.1 Content Features
- [x] Code syntax highlighting (via Glamour/Chroma)
- [ ] Image support:
  - [ ] ASCII art conversion for terminal
  - [ ] Normal images in web view
- [ ] Post series/collections
- [ ] Related posts suggestions
- [x] Reading time estimates
- [ ] Table of contents generation

#### 6.2 Engagement Features
- [ ] Simple view counter
- [ ] "Newsletter" via RSS (with instructions page)
- [ ] Webmention support
- [ ] Comment system (optional, maybe via GitHub issues)

#### 6.3 Developer Experience
- [x] Hot reload in development (file watcher)
- [x] Docker image (multi-stage build)
- [x] Docker Compose with Caddy example (commented)
- [ ] Systemd service file
- [ ] Ansible playbook for deployment
- [x] GitHub Actions for CI/CD (release workflow)
- [x] Health check endpoint (/health)

### Phase 7: Documentation & Polish 🔶 PARTIAL

#### 7.1 User Documentation
- [x] README with quick start
- [x] Installation guide (binary, Docker, source)
- [x] Configuration reference (docs/CONFIGURATION.md)
- [x] Theme creation guide (docs/THEMES.md)
- [x] Release process guide (docs/RELEASING.md)
- [ ] Migration guides (from Hugo, Jekyll, etc.)

#### 7.2 Final Polish
- [x] Basic error handling
- [ ] Security audit (SSH, input validation)
- [ ] Performance optimization
- [ ] Accessibility review (screen reader support in web)
- [ ] Cross-terminal testing (iTerm, Alacritty, Windows Terminal, etc.)

---

## Milestones & Suggested Order

### MVP (Minimum Viable Product) ✅ COMPLETE
1. Phase 1.1-1.3: Core infrastructure
2. Phase 2.1-2.2: Basic TUI reader
3. Phase 3.1: SSH server
4. Phase 1.2 (RSS part): RSS feed

**MVP Deliverable**: SSH into your blog, browse posts, read them, get RSS feed.

### Beta Release ✅ COMPLETE
5. Phase 4.1-4.3: Web interface with xterm.js
6. Phase 2.3: Visual polish
7. Phase 5.3: CLI management tool

**Beta Deliverable**: Full reader experience via SSH and web, basic content management.

### 1.0 Release 🔶 IN PROGRESS
8. Phase 5.1-5.2: Full admin interface (NOT STARTED)
9. Phase 6.1-6.2: Advanced content features (PARTIAL)
10. Phase 7: Documentation and polish (PARTIAL)

---

## Current Project Status (as of 2026-02-05)

### Completed Features
- Full TUI reader with 8 built-in themes
- SSH server with rate limiting, MOTD, non-interactive commands
- Web terminal with xterm.js, theme sync, reconnection
- Static HTML pages (archive, posts, tags) with SEO tags
- RSS/Atom/JSON feeds and sitemap
- CLI tools: new, publish, unpublish, delete, list, schedule, sync, serve
- Docker + Docker Compose support
- GitHub Actions release workflow
- File watcher for auto-sync
- Theme persistence per SSH fingerprint

### Remaining Work for 1.0

**High Priority:**
- [ ] Admin TUI interface (create/edit/delete posts via SSH)
- [ ] Import tool for Hugo/Jekyll migration
- [ ] View counter/analytics

**Medium Priority:**
- [ ] Full-text search (FTS5) instead of LIKE
- [ ] Tag filtering in TUI
- [ ] Table of contents generation
- [ ] RSS instructions page

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
| Phase 1 | Medium | ✅ Complete |
| Phase 2 | High | ✅ Complete |
| Phase 3 | Low | ✅ Complete |
| Phase 4 | Medium | ✅ Complete |
| Phase 5 | Medium | 🔶 CLI done, TUI/Web admin not started |
| Phase 6 | Variable | 🔶 Partial (syntax highlighting, reading time) |
| Phase 7 | Low | 🔶 Partial (docs exist, polish remaining) |

---

## Open Questions to Decide

1. **Editor Integration**: Should admin TUI launch $EDITOR (vim/nano) or have built-in editing?
2. **Analytics**: Simple view counts, or integrate with something like Plausible?
3. **Comments**: Skip entirely, GitHub Issues integration, or custom?
4. **Multi-user**: Single author blog or support multiple authors?
5. **Custom Domain per Blog**: Single blog instance or multi-tenant?
