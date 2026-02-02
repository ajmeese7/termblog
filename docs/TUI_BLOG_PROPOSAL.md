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

### Phase 1: Core Infrastructure (Foundation)

#### 1.1 Project Setup
- [ ] Initialize Go module (`go mod init github.com/ajmeese7/termblog`)
- [ ] Set up directory structure:
  ```
  termblog/
  ├── cmd/
  │   └── termblog/
  │       └── main.go
  ├── internal/
  │   ├── blog/           # Core blog logic
  │   ├── ssh/            # SSH server (Wish)
  │   ├── web/            # HTTP server
  │   ├── tui/            # Bubbletea models
  │   ├── storage/        # SQLite + filesystem
  │   └── theme/          # Terminal themes
  ├── content/
  │   └── posts/          # Markdown files
  ├── web/
  │   ├── static/         # CSS, JS (xterm.js)
  │   └── templates/      # HTML templates
  ├── themes/             # Terminal color schemes
  └── config.yaml
  ```
- [ ] Add core dependencies:
  - `github.com/charmbracelet/bubbletea`
  - `github.com/charmbracelet/wish`
  - `github.com/charmbracelet/lipgloss`
  - `github.com/charmbracelet/glamour`
  - `github.com/mattn/go-sqlite3`
  - `gopkg.in/yaml.v3`
- [ ] Create configuration system (YAML-based)
- [ ] Set up basic logging

#### 1.2 Storage Layer
- [ ] Design SQLite schema:
  ```sql
  -- Posts metadata (content in filesystem)
  CREATE TABLE posts (
    id TEXT PRIMARY KEY,
    slug TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    created_at DATETIME,
    updated_at DATETIME,
    published_at DATETIME,
    status TEXT CHECK(status IN ('draft', 'published', 'scheduled')),
    tags TEXT,  -- JSON array
    filepath TEXT NOT NULL
  );

  -- Themes
  CREATE TABLE themes (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    config TEXT NOT NULL  -- JSON: colors, styles
  );

  -- Settings
  CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT
  );
  ```
- [ ] Implement post repository (CRUD operations)
- [ ] Implement filesystem watcher for hot-reload of markdown files
- [ ] Build markdown parser with frontmatter support:
  ```yaml
  ---
  title: My Post
  date: 2026-01-28
  tags: [go, tui, blog]
  status: published
  ---
  # Content here...
  ```

#### 1.3 Blog Engine Core
- [ ] Post listing with pagination
- [ ] Post retrieval by slug
- [ ] Tag-based filtering
- [ ] Full-text search (SQLite FTS5)
- [ ] RSS/Atom feed generation
- [ ] Sitemap generation

### Phase 2: TUI Reader Interface

#### 2.1 Bubbletea Models
- [ ] **Main Menu Model**: Navigation hub
  - Recent posts list
  - Search prompt
  - Theme selector
  - About/RSS info
- [ ] **Post List Model**: Paginated, filterable post listing
  - Vim-style navigation (j/k/gg/G)
  - Tag filtering
  - Sort options (date, title)
- [ ] **Post Reader Model**: Full post display
  - Glamour markdown rendering
  - Scrolling (mouse + keyboard)
  - Link extraction and display
  - "Back to list" navigation
- [ ] **Search Model**: Interactive search
  - Real-time filtering
  - Search in titles + content
  - Highlight matches
- [ ] **Theme Selector Model**: Live theme preview
  - Cycle through installed themes
  - Instant preview

#### 2.2 Terminal Themes System
- [ ] Design theme specification format (JSON/YAML):
  ```yaml
  name: "Pip-Boy Green"
  colors:
    background: "#0a0a0a"
    foreground: "#00ff00"
    accent: "#00cc00"
    muted: "#006600"
    border: "#00ff00"
  styles:
    title: bold
    date: italic
    tag: reverse
  ```
- [ ] Implement built-in themes:
  - [ ] **Monochrome**: Pure black and white
  - [ ] **Pip-Boy**: Green phosphor CRT aesthetic
  - [ ] **Amber**: Amber monochrome terminal
  - [ ] **Matrix**: Green on black, "digital rain" feel
  - [ ] **Paper**: Light theme, like thermal printer output
- [ ] Theme persistence (remember user choice via SSH key fingerprint)
- [ ] Theme hot-reload for development

#### 2.3 Visual Polish
- [ ] ASCII art header/logo support
- [ ] Animated transitions (subtle, optional)
- [ ] Loading spinners for slow operations
- [ ] Box drawing characters for borders
- [ ] Status bar with current location/time

### Phase 3: SSH Server

#### 3.1 Wish Integration
- [ ] Basic SSH server setup with Wish
- [ ] Host key generation and management
- [ ] Banner/MOTD display on connect
- [ ] Graceful connection handling
- [ ] Rate limiting (prevent abuse)
- [ ] Connection logging

#### 3.2 Authentication (Optional)
- [ ] Public key authentication for admin access
- [ ] Anonymous read access (default)
- [ ] `~/.ssh/authorized_keys` integration

#### 3.3 SSH Commands (Non-TUI Mode)
- [ ] `ssh blog.yoursite.com posts` - List posts as plain text
- [ ] `ssh blog.yoursite.com read <slug>` - Output post to stdout
- [ ] `ssh blog.yoursite.com rss` - Output RSS feed
- [ ] `ssh blog.yoursite.com search <query>` - Search posts

### Phase 4: Web Interface

#### 4.1 Web Terminal (xterm.js)
- [ ] Minimal HTML page that loads xterm.js
- [ ] WebSocket proxy to SSH/TUI backend
- [ ] Mobile-responsive terminal sizing
- [ ] Touch support for mobile
- [ ] Copy/paste support
- [ ] Connection status indicator
- [ ] Reconnection logic

#### 4.2 Static HTML Fallback
- [ ] Minimal HTML templates for:
  - [ ] Homepage/post list
  - [ ] Individual post pages
  - [ ] Tag pages
  - [ ] RSS feed page (with subscription instructions)
- [ ] Terminal-inspired CSS theme (optional)
- [ ] Zero JavaScript requirement
- [ ] Proper semantic HTML for accessibility
- [ ] OpenGraph/Twitter meta tags

#### 4.3 HTTP Server
- [ ] Route handling:
  - `GET /` - Web terminal or post list
  - `GET /posts/:slug` - Post page
  - `GET /tags/:tag` - Tag listing
  - `GET /feed.xml` - RSS feed
  - `GET /feed.json` - JSON feed
  - `GET /sitemap.xml` - Sitemap
  - `GET /ws` - WebSocket for terminal
- [ ] Content negotiation (HTML vs JSON API)
- [ ] Static file serving
- [ ] Gzip compression
- [ ] Cache headers

### Phase 5: Admin Interface

#### 5.1 Admin TUI (via SSH)
- [ ] Authenticate via SSH key
- [ ] Post management:
  - [ ] Create new post (opens $EDITOR or built-in)
  - [ ] Edit existing post
  - [ ] Delete post (with confirmation)
  - [ ] Publish/unpublish toggle
  - [ ] Schedule post for future
- [ ] Theme management:
  - [ ] Upload new theme
  - [ ] Set default theme
  - [ ] Delete theme
- [ ] Settings:
  - [ ] Blog title, description, author
  - [ ] Posts per page
  - [ ] Enable/disable features
- [ ] Simple analytics (view counts, popular posts)

#### 5.2 Admin Web UI (Alternative)
- [ ] Simple login page (password or SSO)
- [ ] Markdown editor with live preview
- [ ] File upload for images
- [ ] Draft management
- [ ] Settings panel

#### 5.3 CLI Management Tool
- [ ] `termblog new "Post Title"` - Create new post file
- [ ] `termblog publish <slug>` - Publish a draft
- [ ] `termblog serve` - Start server
- [ ] `termblog import <hugo|jekyll|bearblog>` - Import from other platforms

### Phase 6: Advanced Features

#### 6.1 Content Features
- [ ] Code syntax highlighting (in terminal!)
- [ ] Image support:
  - ASCII art conversion for terminal
  - Normal images in web view
- [ ] Post series/collections
- [ ] Related posts suggestions
- [ ] Reading time estimates
- [ ] Table of contents generation

#### 6.2 Engagement Features
- [ ] Simple view counter
- [ ] "Newsletter" via RSS (with instructions)
- [ ] Webmention support
- [ ] Comment system (optional, maybe via GitHub issues)

#### 6.3 Developer Experience
- [ ] Hot reload in development
- [ ] Docker image
- [ ] Docker Compose with Caddy/Traefik
- [ ] Systemd service file
- [ ] Ansible playbook for deployment
- [ ] GitHub Actions for CI/CD
- [ ] Health check endpoint

### Phase 7: Documentation & Polish

#### 7.1 User Documentation
- [ ] README with quick start
- [ ] Installation guide (binary, Docker, source)
- [ ] Configuration reference
- [ ] Theme creation guide
- [ ] Migration guides (from Hugo, Jekyll, etc.)

#### 7.2 Final Polish
- [ ] Error handling review
- [ ] Security audit (SSH, input validation)
- [ ] Performance optimization
- [ ] Accessibility review (screen reader support in web)
- [ ] Cross-terminal testing (iTerm, Alacritty, Windows Terminal, etc.)

---

## Milestones & Suggested Order

### MVP (Minimum Viable Product)
1. Phase 1.1-1.3: Core infrastructure
2. Phase 2.1-2.2: Basic TUI reader
3. Phase 3.1: SSH server
4. Phase 1.2 (RSS part): RSS feed

**MVP Deliverable**: SSH into your blog, browse posts, read them, get RSS feed.

### Beta Release
5. Phase 4.1-4.3: Web interface with xterm.js
6. Phase 2.3: Visual polish
7. Phase 5.3: CLI management tool

**Beta Deliverable**: Full reader experience via SSH and web, basic content management.

### 1.0 Release
8. Phase 5.1-5.2: Full admin interface
9. Phase 6.1-6.2: Advanced content features
10. Phase 7: Documentation and polish

---

## Estimated Complexity

| Phase | Complexity | Key Challenges |
|-------|------------|----------------|
| Phase 1 | Medium | SQLite schema design, config system |
| Phase 2 | High | Bubbletea state management, theme system |
| Phase 3 | Low | Wish makes this straightforward |
| Phase 4 | Medium | WebSocket proxy, mobile support |
| Phase 5 | Medium | Editor integration, file management |
| Phase 6 | Variable | Syntax highlighting in terminal is tricky |
| Phase 7 | Low | Documentation writing |

---

## Open Questions to Decide

1. **Editor Integration**: Should admin TUI launch $EDITOR (vim/nano) or have built-in editing?
2. **Analytics**: Simple view counts, or integrate with something like Plausible?
3. **Comments**: Skip entirely, GitHub Issues integration, or custom?
4. **Multi-user**: Single author blog or support multiple authors?
5. **Custom Domain per Blog**: Single blog instance or multi-tenant?
