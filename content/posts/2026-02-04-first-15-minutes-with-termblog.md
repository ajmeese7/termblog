---
title: "Your First 15 Minutes with TermBlog"
description: "A practical quick start from binary install to your first published post"
author: "TermBlog Team"
date: 2026-02-04
tags: ["quickstart", "setup", "guide"]
draft: false
---

# Your First 15 Minutes with TermBlog

This guide walks through a fast, realistic setup path.

## 1. Install and run

```bash
git clone https://github.com/ajmeese7/termblog.git
cd termblog
make build
./termblog serve
```

By default, the app serves:

- SSH on port `2222`
- HTTP on port `8080`

## 2. Open the app

In another terminal:

```bash
ssh localhost -p 2222
```

Or in a browser:

```text
http://localhost:8080
```

## 3. Create a real post

```bash
./termblog new "Shipping our terminal blog"
```

Edit the generated Markdown file in `content/posts/`, then make sure frontmatter includes:

```yaml
draft: false
```

## 4. Sync content

```bash
./termblog sync
```

When `serve` is running, file changes are also watched and synced automatically.

## 5. Validate navigation

Use these keys in the UI:

- `j` / `k` to move
- `enter` to open
- `/` to search
- `t` to change theme
- `?` for help

## 6. Prep for deployment

Before going live, set `base_url` in `config.yaml`:

```yaml
blog:
  base_url: "https://termblog.com"
```

Now your feeds and generated links point to the production domain.

## Next step

Once this flow feels good locally, move the same content directory and config to your server. The local workflow is the production workflow.
