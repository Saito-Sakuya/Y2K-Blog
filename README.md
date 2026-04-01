<div align="center">
  <img src="docs/assets/y982blog_logo.png" alt="Y982Blog Logo" width="160"/>
  <h1>Y982Blog (Y2K Pixel Blog)</h1>
  <p>English | <strong>[中文](README_ZH.md)</strong></p>
  <p><em>Break free from the endless scroll. Welcome back to the cyber-desktop of the Y2K era.</em></p>
</div>

## Why is it different?

In a sea of identical WordPress templates, Hexo generators, and modern minimalist themes, **Y982Blog** chooses a path of rebellious nostalgia:

🚀 **Reject the Infinite Scroll**: We've completely abandoned the monotonous vertical scrolling of modern web design. Your entire blog is an interactive canvas filled with dynamic particle effects. Every article, photo gallery, and rating board opens as a **fully draggable, resizable Windows 98 window**. Readers can open multiple posts simultaneously and explore your digital garden just like a real operating system!

🔍 **Native Blazing-Fast Search**: No heavy Elasticsearch cluster required! Y982Blog leverages PostgreSQL's latent power, using GIN indexes and the `pg_trgm` extension to deliver instantaneous full-text trigram search. Query via `#tags` or `~fulltext` syntax instantly.

💻 **Built for Performance & Scale**:
- **Frontend (Next.js 16)**: SSR for flawless SEO, paired with global state management to handle complex desktop Z-index window stacking.
- **Admin Panel (Vite + 98.css)**: A completely decoupled SPA administration dashboard delivering a pure, unfiltered Win98 visual aesthetic.
- **Backend (Go 1.23 + Gin)**: A hyper-lightweight, high-concurrency core featuring an onboard zero-config Let's Encrypt automated SSL issuing system for independent frontend and backend domains.

> **Open Source**: Released freely under the [MIT License](LICENSE). We provide the bleeding-edge retro framework—you inject the soul.

## Demo

### Particle Background & Window Animations
![Particle System Demo](docs/assets/particle_demo.webp)

### Frontend (Desktop UI)
![Frontend Demo](docs/assets/frontend_demo.webp)

### Admin Panel (98.css)
![Admin Demo](docs/assets/admin_demo.webp)

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                   Nginx (:80/:443)                   │
│  blog.example.com  │  admin.example.com  │  /api/*   │
└─────────┬──────────┴──────────┬──────────┴─────┬─────┘
          │                     │                │
    ┌─────▼─────┐        ┌─────▼─────┐    ┌─────▼─────┐
    │ Frontend  │        │   Admin   │    │  Go API   │
    │ Next.js   │        │ Vite SPA  │    │  Gin      │
    │ :3000     │        │ :80       │    │  :8080    │
    └───────────┘        └───────────┘    └─────┬─────┘
                                                │
                                          ┌─────▼─────┐
                                          │PostgreSQL │
                                          │ :5432     │
                                          └───────────┘
```

## Project Structure

```
.
├── frontend/          Next.js 16 — public-facing blog
│   ├── app/           App Router pages
│   ├── components/    React components (Window, Spotlight, Taskbar, etc.)
│   ├── lib/           API client, utilities
│   └── public/        Static assets
├── admin/             Vite + React — admin panel (98.css UI)
│   └── src/
│       ├── pages/     Dashboard, PostEditor, Settings, Login
│       ├── api/       Axios-based API client
│       └── context/   Auth context (JWT)
├── backend/           Go + Gin — REST API server
│   ├── cmd/server/    Entry point, routing, server startup
│   ├── internal/
│   │   ├── handler/   HTTP handlers (34 routes)
│   │   ├── service/   Business logic
│   │   ├── repository/ Database queries
│   │   ├── model/     Data models
│   │   └── middleware/ JWT auth, CORS
│   └── migrations/    PostgreSQL schema migrations (001–009)
├── docker-compose.yml 5-service production orchestration
├── nginx.conf         Multi-domain reverse proxy config
├── .env.example       Environment variable template
└── README.md
```

## Quick Start (Development)

### Prerequisites

- Go 1.23+
- Node.js 20+
- Docker (for PostgreSQL)

### Steps

```bash
# 1. Start PostgreSQL
docker compose up db -d

# 2. Start the backend API
cd backend
cp .env.example .env        # edit as needed
go run ./cmd/server/         # → http://localhost:8080

# 3. Start the frontend (separate terminal)
cd frontend
npm install
npm run dev                  # → http://localhost:3000

# 4. Start the admin panel (separate terminal)
cd admin
npm install
npm run dev                  # → http://localhost:5173
```

On first launch, visit the admin panel and follow the setup wizard to create an admin account.

## Production Deployment

```bash
# 1. Configure environment
cp .env.example .env
# Edit .env: set JWT_SECRET, DB_PASSWORD, etc.
# Generate a secret: openssl rand -base64 32

# 2. Edit nginx.conf
# Replace example.com with your actual domains

# 3. Build and start all services
docker compose up -d --build

# 4. Configure domains and SSL
# Open the admin panel → Settings → Domain & SSL
# Set frontend/admin domains
# Choose SSL mode: auto (Let's Encrypt) or manual (PEM upload)
```

### Docker Services

| Service    | Image              | Role                                 |
|------------|--------------------|--------------------------------------|
| `db`       | postgres:16-alpine | Database with pg_trgm extension      |
| `api`      | Go 1.23 (custom)   | REST API, SSL termination            |
| `frontend` | Node 20 (custom)   | Next.js standalone SSR server        |
| `admin`    | Nginx Alpine       | Vite-built SPA with SPA fallback     |
| `nginx`    | nginx:alpine       | Reverse proxy, rate limiting, caching|

## Environment Variables

| Variable       | Required | Default          | Description                            |
|----------------|----------|------------------|----------------------------------------|
| `JWT_SECRET`   | Yes      | —                | JWT signing secret                     |
| `DB_USER`      |          | `blog`           | PostgreSQL username                    |
| `DB_PASSWORD`  |          | `blog`           | PostgreSQL password                    |
| `API_URL`      |          | `http://api:8080`| Backend URL for frontend SSR           |
| `ADMIN_API_URL`|          | `/api`           | Backend URL for admin panel            |
| `AI_API_URL`   |          | —                | OpenAI-compatible API endpoint         |
| `AI_API_KEY`   |          | —                | AI API key                             |
| `AI_MODEL`     |          | `deepseek-chat`  | Model name for AI summaries            |

Domain names and SSL settings are configured through the admin panel, not environment variables.

## API

The backend exposes 34 REST endpoints across three groups:

### Public

| Method | Path                | Description                         |
|--------|---------------------|-------------------------------------|
| GET    | `/api/boards`       | Board tree                          |
| GET    | `/api/boards/:slug` | Board contents (paginated, sorted)  |
| GET    | `/api/posts/:slug`  | Single post (article/photo/rating/page) |
| GET    | `/api/search?q=`    | Search (default / `#tag` / `~fulltext`) |
| GET    | `/api/tags`         | All tags with counts                |
| GET    | `/api/menu`         | Navigation menu data                |
| GET    | `/api/ai/summary`   | AI-generated summary (cached)       |
| GET    | `/api/og/:slug`     | Open Graph metadata                 |
| GET    | `/api/preview/:token` | Token-authenticated preview       |
| GET    | `/api/css-config`   | Custom CSS configuration            |
| GET    | `/feed.xml`         | RSS 2.0 feed                        |
| GET    | `/sitemap.xml`      | XML sitemap                         |
| GET    | `/robots.txt`       | Robots directives                   |

### Authentication

| Method | Path                       | Description              |
|--------|----------------------------|--------------------------|
| POST   | `/api/admin/login`         | Login (bcrypt + captcha) |
| GET    | `/api/admin/captcha`       | Math captcha challenge   |
| GET    | `/api/admin/login/status`  | IP ban status            |
| GET    | `/api/setup/status`        | First-run check          |
| POST   | `/api/setup/initialize`    | Create initial admin     |

### Admin (JWT required)

| Method | Path                             | Description             |
|--------|----------------------------------|-------------------------|
| GET    | `/api/admin/posts`               | List posts by status    |
| POST   | `/api/admin/posts`               | Create post             |
| PUT    | `/api/admin/posts/:slug`         | Update post             |
| DELETE | `/api/admin/posts/:slug`         | Permanently delete post |
| POST   | `/api/admin/posts/:slug/trash`   | Move to trash           |
| POST   | `/api/admin/posts/:slug/restore` | Restore from trash      |
| DELETE | `/api/admin/trash`               | Empty trash             |
| POST   | `/api/admin/preview/:slug`       | Generate preview token  |
| POST   | `/api/admin/boards`              | Create/update board     |
| DELETE | `/api/admin/ai-cache/:slug`      | Clear AI cache          |
| PUT    | `/api/admin/password`            | Change password         |
| GET    | `/api/admin/settings`            | Get site settings       |
| PUT    | `/api/admin/settings`            | Update settings         |
| PUT    | `/api/admin/ssl`                 | Upload SSL certificate  |
| DELETE | `/api/admin/ssl`                 | Remove SSL certificate  |

## Content Types

| Type    | Slug prefix | Description                                           |
|---------|-------------|-------------------------------------------------------|
| Article | —           | Standard blog post with Markdown body                 |
| Photo   | —           | Multi-page photo essay (left image, right text)       |
| Rating  | —           | Review with cover, radar charts, AI summary, and body |
| Page    | —           | Static page (About, Links, etc.), optionally in menu  |

All content is stored in PostgreSQL. Posts support draft, published, and trashed states.

## Search

Search uses PostgreSQL's `pg_trgm` extension with GIN indexes for trigram-based matching. This provides native support for Chinese, Japanese, and Korean text without a dedicated tokenizer.

| Prefix | Scope                         | Example      |
|--------|-------------------------------|--------------|
| (none) | Title, tags, excerpt          | `pixel art`  |
| `#`    | Tags only                     | `#design`    |
| `~`    | Full text (title + body + tags) | `~particles` |

Results are ranked by `similarity()` score.

## SSL/TLS

SSL is managed per-domain through the admin panel (Settings > Domain & SSL). Each domain (frontend and admin) can be configured independently.

| Mode   | Behavior                                                    |
|--------|-------------------------------------------------------------|
| Off    | No SSL. Backend listens on `:8080` HTTP only.               |
| Manual | Upload PEM certificate and private key via admin panel.     |
| Auto   | Let's Encrypt via `autocert`. Requires port 80 from public internet. |

When any domain has SSL enabled, the backend starts HTTPS on `:443` and HTTP on `:80` (for ACME challenges and HTTP-to-HTTPS redirects). Certificates are dispatched by SNI (Server Name Indication).

## Security

- **Password hashing**: bcrypt
- **Authentication**: JWT (HS256, 24h expiry)
- **Brute-force protection**: Math captcha on login, per-IP failure counter, automatic 15-minute ban after 10 failed attempts
- **Input sanitization**: Markdown only (no raw HTML), DOMPurify on render
- **SQL injection**: Parameterized queries throughout
- **SSL keys**: Stored in database, never returned via API (only `hasCert` boolean)
- **AI keys**: Stored server-side, frontend sees only masked values

## Tech Stack

| Component | Technology                                  |
|-----------|---------------------------------------------|
| Frontend  | Next.js 16, React 19, TypeScript, zustand   |
| Admin     | Vite 5, React 19, TypeScript, 98.css, axios |
| Backend   | Go 1.23, Gin, PostgreSQL 16                 |
| Search    | pg_trgm (trigram indexes, CJK support)      |
| SSL       | autocert (Let's Encrypt) + manual PEM + SNI |
| Deploy    | Docker Compose, Nginx reverse proxy         |
| SEO       | RSS 2.0, XML Sitemap, Open Graph, robots.txt|

## Database Migrations

Migrations are applied automatically when the PostgreSQL container starts (via `docker-entrypoint-initdb.d`). For existing databases, apply new migrations manually:

```bash
docker cp backend/migrations/009_chinese_search.sql y2k-blog-db:/tmp/
docker exec y2k-blog-db psql -U blog -d y2k_blog -f /tmp/009_chinese_search.sql
```

## License

[MIT](LICENSE)
