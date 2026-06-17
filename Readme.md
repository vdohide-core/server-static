# server-static

Content delivery server for vdohide-core — proxies HLS streams, images, sprites, and thumbnails from storage nodes.

---

## Features

- HLS master playlist (`/{slug}/playlist.m3u8`)
- HLS segment playlist (`/{mediaSlug}/video.m3u8`) with CDN domain rewriting
- Image proxy with on-the-fly resize (`?w=400&h=300&fit=cover&q=80`)
- Sprite sheet + VTT proxy (`/{slug}/sprite/sprite.vtt`, `/{slug}/sprite/{n}.jpg`)
- Thumbnail poster proxy (`/thumb/{slug}/{n}.jpg`)
- Image not-found PNG placeholder
- Rotating log file (25 MB per file, file-only — no stdout noise)
- Log reader API (`GET /logs`, `GET /logs/{filename}?tail=200`)

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8082` | HTTP listen port |
| `MONGODB_URI` | _(required)_ | MongoDB connection string |
| `LOG_PATH` | `logs/server-static.log` | Log file path |

---

## Install

### 1 domain

```bash
curl -fsSL https://raw.githubusercontent.com/vdohide-core/server-static/main/install.sh | sudo -E bash -s -- \
    --port 8082 \
    --domain cdn.vdohide.com \
    --mongodb-uri "mongodb+srv://user:pass@host/platform"
```

### Multiple domains (1, 2, 3, … N)

```bash
curl -fsSL https://raw.githubusercontent.com/vdohide-core/server-static/main/install.sh | sudo -E bash -s -- \
    --port 8082 \
    --domain cdn.vdohide.com cdn2.vdohide.com cdn3.vdohide.com \
    --mongodb-uri "mongodb+srv://user:pass@host/platform"
```

### App only (no Nginx)

```bash
curl -fsSL https://raw.githubusercontent.com/vdohide-core/server-static/main/install.sh | sudo -E bash -s -- \
    --app \
    --port 8082 \
    --mongodb-uri "mongodb+srv://user:pass@host/platform"
```

### Nginx only (add more domains to existing install)

```bash
curl -fsSL https://raw.githubusercontent.com/vdohide-core/server-static/main/install.sh | sudo -E bash -s -- \
    --nginx \
    --port 8082 \
    --domain cdn4.vdohide.com cdn5.vdohide.com
```

### Uninstall

```bash
curl -fsSL https://raw.githubusercontent.com/vdohide-core/server-static/main/install.sh | sudo bash -s -- --uninstall
```

---

## API Routes

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check |
| `GET` | `/{slug}/playlist.m3u8` | HLS master playlist |
| `GET` | `/{mediaSlug}/video.m3u8` | HLS segment playlist |
| `GET` | `/{slug}/sprite/sprite.vtt` | Sprite VTT |
| `GET` | `/{slug}/sprite/{n}.jpg` | Sprite image |
| `GET` | `/thumb/{slug}/{n}.jpg` | Thumbnail poster |
| `GET` | `/{slug}.{ext}` | File stream / image proxy |
| `GET` | `/logs` | List log files |
| `GET` | `/logs/{filename}?tail=200` | Read log file (newest first) |

---

## Service Management

```bash
systemctl status  server-static
systemctl restart server-static
systemctl stop    server-static
journalctl -u server-static -f
```

---

## Release

```bash
git tag v1.0.1
git push origin v1.0.1
```
