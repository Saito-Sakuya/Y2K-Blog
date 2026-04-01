# Y982Blog Quick Deployment Guide

## Prerequisites

- Docker Engine 24+ and Docker Compose V2
- A server with ports 80 and 443 open (for SSL)
- A domain name pointing to your server IP

## Steps

```bash
# 1. Configure environment variables
cp .env.example .env
#    Edit .env:
#      - Set JWT_SECRET (generate: openssl rand -base64 32)
#      - Set DB_PASSWORD
#      - Set AI_API_KEY and AI_API_URL (optional, for AI summaries)

# 2. Configure Nginx domains
#    Edit nginx.conf:
#      - Replace blog.example.com with your frontend domain
#      - Replace admin.example.com with your admin domain

# 3. Start all services
docker compose up -d
```

Database migrations are automatically extracted from the API image on first launch.

## First Launch

1. Open `http://your-admin-domain` in your browser
2. Follow the setup wizard to create an admin account
3. Go to Settings to configure site title, domains, and SSL

## SSL Configuration

SSL is managed through the admin panel (Settings > Domain & SSL):

| Mode   | Description                                                              |
|--------|--------------------------------------------------------------------------|
| Off    | HTTP only                                                                |
| Auto   | Automatic Let's Encrypt certificate. Requires port 80 open to internet. |
| Manual | Upload your own PEM certificate and private key.                         |

## Updating

```bash
# Pull latest images and restart
docker compose pull
docker compose up -d
```

## Useful Commands

```bash
# View logs
docker compose logs -f

# Stop all services
docker compose down

# Reset database (WARNING: deletes all data)
docker compose down -v
```

## More Information

- Full documentation: https://github.com/Saito-Sakuya/Y2K-Blog
- Operation manual (Chinese): https://github.com/Saito-Sakuya/Y2K-Blog/blob/master/MANUAL_ZH.md
