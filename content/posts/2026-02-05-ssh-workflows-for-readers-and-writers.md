---
title: "SSH Workflows for Readers and Writers"
description: "How to use TermBlog as an interactive app and as a command-line interface"
author: "TermBlog Team"
date: 2026-02-05
tags: ["ssh", "workflow", "automation"]
draft: false
---

# SSH Workflows for Readers and Writers

Most people see SSH and think remote shell access. In TermBlog, SSH is also your content API.

## Interactive mode

Standard SSH gives you the full terminal app:

```bash
ssh termblog.com -p 2222
```

This is ideal for browsing posts, searching tags, and reading in a low-latency interface.

## Command mode

TermBlog also supports non-interactive commands over SSH.

```bash
ssh termblog.com posts
ssh termblog.com read welcome-to-termblog
ssh termblog.com search terminal
ssh termblog.com rss > feed.xml
```

This enables lightweight automation without adding another HTTP auth surface.

## Example: morning digest script

```bash
#!/usr/bin/env bash
set -euo pipefail

ssh termblog.com posts | head -n 10
```

## Why this model works

- Human readers get a complete app.
- Scripts get direct command output.
- Operators manage fewer moving parts.

One protocol, two use cases.

## Good defaults for production

- Keep SSH on a non-default port.
- Use key-based auth for admin access.
- Keep the post workflow file-based and version controlled.

TermBlog does not fight terminal-native habits. It leans into them.
