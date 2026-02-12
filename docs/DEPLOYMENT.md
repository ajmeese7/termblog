# Deployment Guide

This guide walks through a complete production setup for launching TermBlog on a real domain (for example, `termblog.com`).

## Deployment Model

TermBlog runs as a single process that serves:
- SSH terminal UI (default `2222`)
- HTTP web terminal and feeds (default `8080`)

In production, a typical setup is:
1. TermBlog runs on a Linux VM via `systemd`
2. A reverse proxy terminates TLS and forwards traffic to `127.0.0.1:8080`
3. DNS points your domain to the server
4. SSH port `2222` remains open for terminal clients

## Prerequisites

- A Linux server with public IP
- A domain you control
- `sudo` access on the server
- Basic SSH familiarity
- Go toolchain (only required for source builds)

## 1. Provision the Server

```bash
sudo adduser --system --group --home /opt/termblog termblog
sudo mkdir -p /opt/termblog
sudo chown -R termblog:termblog /opt/termblog
```

## 2. Install TermBlog

Choose one method.

### Option A: Download release binary

```bash
cd /tmp
curl -sSL https://github.com/ajmeese7/termblog/releases/latest/download/termblog_*_linux_amd64.tar.gz | tar xz
sudo install -m 0755 termblog /opt/termblog/termblog
```

### Option B: Build from source

Install Go first (Ubuntu/Debian example):

```bash
sudo apt update
sudo apt install -y golang-go make git
go version
```

Then build and install TermBlog:

```bash
git clone https://github.com/ajmeese7/termblog.git
cd termblog
make build
sudo install -m 0755 termblog /opt/termblog/termblog
```

## 3. Copy Config and Content

Choose the copy method that matches your setup.

### Option A: Copy from a different machine (remote -> server)

```bash
rsync -avz config.yaml content/ user@your-server:/opt/termblog/
```

### Option B: Copy on the same server (local clone -> /opt/termblog)

If you cloned `termblog` directly on the target server, copy locally with `sudo`:

```bash
sudo rsync -av --chown=termblog:termblog config.yaml content/ /opt/termblog/
```

Why: `/opt/termblog` is owned by the `termblog` service user, so running `rsync` without `sudo` will fail with `Permission denied`.
Using `content/` (not `content/posts/`) preserves the expected path at `/opt/termblog/content/posts`.

Do not use `termblog@host` over SSH for this step. The `termblog` service user is typically non-login and not intended for interactive authentication.

On the server, ensure ownership:

```bash
sudo chown -R termblog:termblog /opt/termblog
```

## 4. Set Production Configuration

Edit `/opt/termblog/config.yaml`:

```yaml
blog:
  title: "TermBlog"
  description: "A blog you can read in your terminal"
  author: "Your Name"
  base_url: "https://termblog.com"
  content_dir: "content/posts"

server:
  ssh_port: 2222
  http_port: 8080
  host_key_path: ".ssh/termblog_host_key"

storage:
  database_path: "termblog.db"
```

Important:
- Set `blog.base_url` to your real domain with `https://`
- Keep `content_dir` and `database_path` writable by `termblog`

## 5. Install and Start systemd Service

The repo includes a service file at `deploy/termblog.service`.

```bash
sudo install -m 0644 deploy/termblog.service /etc/systemd/system/termblog.service
sudo systemctl daemon-reload
sudo systemctl enable --now termblog
sudo systemctl status termblog
```

Check logs:

```bash
sudo journalctl -u termblog -f
```

## 6. Configure TLS Reverse Proxy

Use any reverse proxy that can terminate TLS and forward to `127.0.0.1:8080`.

### Option A: Caddy

Caddy provides automatic certificate management by default.

`/etc/caddy/Caddyfile`:

```caddy
termblog.com, www.termblog.com {
    encode zstd gzip
    reverse_proxy 127.0.0.1:8080
}
```

Apply:

```bash
sudo systemctl reload caddy
```

### Option B: Nginx

Use Nginx as TLS terminator and reverse proxy to TermBlog's HTTP port.

`/etc/nginx/sites-available/termblog.com`:

```nginx
server {
    listen 80;
    listen [::]:80;
    server_name termblog.com www.termblog.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;
    server_name termblog.com www.termblog.com;

    ssl_certificate /etc/letsencrypt/live/termblog.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/termblog.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Required for browser terminal WebSocket /ws
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 86400;
    }
}
```

Enable and reload:

```bash
sudo ln -s /etc/nginx/sites-available/termblog.com /etc/nginx/sites-enabled/termblog.com
sudo nginx -t
sudo systemctl reload nginx
```

For TLS certificates, issue a cert for `termblog.com` and `www.termblog.com` (for example with Let's Encrypt / Certbot) before reloading Nginx.

## 7. DNS Records

At your DNS provider:
- `A` record: `termblog.com` -> your server IPv4
- `CNAME` record: `www.termblog.com` -> `termblog.com`
- `AAAA` records if using IPv6

Wait for propagation, then verify:

```bash
dig +short termblog.com
dig +short www.termblog.com
```

## 8. Firewall and Ports

Allow:
- `22/tcp` (or your custom SSH admin port)
- `80/tcp` and `443/tcp` (HTTP/HTTPS)
- `2222/tcp` (TermBlog SSH reader port)

### Option A: `ufw`

```bash
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 2222/tcp
sudo ufw enable
```

### Option B: `iptables` (Debian/Ubuntu example)

```bash
sudo iptables -A INPUT -p tcp --dport 22 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 80 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 2222 -j ACCEPT
```

Persist rules across reboot:

```bash
sudo apt install -y iptables-persistent
sudo netfilter-persistent save
```

## 9. Final Verification

Run checks from a different machine:

```bash
curl -I https://termblog.com
curl -I https://termblog.com/feed.xml
ssh termblog.com -p 2222
ssh termblog.com posts
```

Expected results:
- HTTPS responds successfully
- Feed endpoint returns `200`
- SSH interactive UI opens on port `2222`
- Non-interactive SSH commands return post output

## 10. Post-Launch Operations

- Backup both:
  - `/opt/termblog/content/posts`
  - `/opt/termblog/termblog.db`
- Monitor service health:

```bash
sudo systemctl status termblog
sudo journalctl -u termblog --since "24 hours ago"
```

## Troubleshooting

- `HTTPS works but no content`: verify Caddy forwards to `127.0.0.1:8080` and TermBlog is running
- `Web terminal is blank on HTTPS`: open browser console; if you see mixed content or `/ws` connection errors, ensure the page uses `wss://` for WebSocket and Nginx includes `Upgrade`/`Connection` headers for `/ws` traffic.
- `SSH to :2222 fails`: check firewall/security group rules and `server.ssh_port`
- `Feed links point to localhost`: fix `blog.base_url` and restart service
- `Posts missing`: verify files exist in `content/posts`, then run `termblog sync` in `/opt/termblog`
