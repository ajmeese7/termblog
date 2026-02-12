---
title: "Launch Checklist for termblog.com"
description: "A practical cut-over checklist for taking TermBlog from local to public production"
author: "TermBlog Team"
date: 2026-02-07
tags: ["launch", "production", "ops"]
draft: false
---

# Launch Checklist for termblog.com

If your content is ready, this checklist gets the service live with minimal surprises.

## 1. Server baseline

- Provision a small Linux VM.
- Create a dedicated `termblog` user.
- Install the binary in `/opt/termblog`.
- Copy `config.yaml` and `content/posts/`.

## 2. Configure production values

In `config.yaml`:

```yaml
blog:
  title: "TermBlog"
  base_url: "https://termblog.com"

server:
  ssh_port: 2222
  http_port: 8080
```

## 3. Run as a service

Use the included unit file at `deploy/termblog.service` and enable it:

```bash
sudo install -m 0644 deploy/termblog.service /etc/systemd/system/termblog.service
sudo systemctl daemon-reload
sudo systemctl enable --now termblog
```

## 4. DNS and edge routing

- Point `termblog.com` and `www.termblog.com` to your server IP.
- Put a reverse proxy in front of port `8080` for TLS.
- Keep SSH (`2222`) reachable for terminal clients.

## 5. Verify externally

Run these checks from a different machine:

```bash
curl -I https://termblog.com
curl -I https://termblog.com/feed.xml
ssh termblog.com -p 2222
```

## 6. Content and feed sanity

- Confirm no key posts are still drafts.
- Check ordering and metadata in the post list.
- Validate feed entries contain correct canonical URLs.

## 7. Post-launch habit

Use a lightweight publish flow:

```bash
termblog new "Post title"
# edit file
termblog publish post-title
termblog sync
```

Keep content in git, deploy often, and ship small updates.
