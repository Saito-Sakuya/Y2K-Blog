# Y2K Pixel Blog — Admin Panel

The admin panel is a standalone single-page application for managing blog content. It runs independently from the public-facing frontend.

## Tech Stack

- React 18 + TypeScript
- Vite 5
- 98.css (Windows 98 retro UI)
- Axios (API client)
- React Router v6

## Development

```bash
npm install
npm run dev   # http://localhost:5173
```

Requires the backend API to be running at `http://localhost:8080`.

## Pages

| Route | Description |
| :--- | :--- |
| `/setup` | First-run setup wizard (redirected automatically) |
| `/login` | Admin login |
| `/dashboard` | Overview: post count, board count, tag count |
| `/posts` | Post list with status filter (published / draft / trash) |
| `/posts/new` | Create post (supports article, rating, photo, page types) |
| `/posts/edit/:slug` | Edit existing post |
| `/boards` | Board management |
| `/settings` | Site settings, AI config, domain & SSL |

## Build

```bash
npm run build   # output to dist/
```

The production build is served as a static SPA via Nginx (see root `docker-compose.yml`).
