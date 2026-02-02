---
title: "Welcome to Termblog"
description: "An introduction to your new terminal-based blog"
author: "Anonymous"
date: 2026-02-01
tags: ["welcome", "introduction"]
draft: false
---

# Welcome to Termblog!

You're reading this post in a terminal. How cool is that?

## What is Termblog?

Termblog is a self-hosted blog platform that lets readers browse and read your posts directly from their terminal using SSH, or through a web-based terminal interface.

## Features

- **SSH Access**: Connect via `ssh blog.example.com -p 2222`
- **Web Terminal**: Access through your browser with xterm.js
- **Vim-style Navigation**: Use `j/k` to move, `enter` to select
- **Markdown Rendering**: Beautiful terminal-rendered markdown
- **RSS Feed**: Subscribe at `/feed.xml`

## Navigation

Here are the key bindings you can use:

| Key | Action |
|-----|--------|
| `j/↓` | Move down |
| `k/↑` | Move up |
| `enter` | Select/Open |
| `esc/h` | Go back |
| `/` | Search |
| `?` | Help |
| `q` | Quit |

## Creating Posts

Create new posts using the CLI:

```bash
termblog new "My New Post"
```

This creates a markdown file in `content/posts/` with frontmatter. Edit the file and set `draft: false` to publish.

## Code Example

Here's some syntax-highlighted code:

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello from Termblog!")
}
```

## What's Next?

1. Explore the interface
2. Create your first post
3. Customize your theme
4. Share your blog URL with the world!

Happy blogging! 🖥️
