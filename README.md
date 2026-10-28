# AdGuard Portal (DoH Identity & Device Gate)

A self-service device onboarding and DNS-over-HTTPS (DoH) access portal for **AdGuard Home**, authenticated via standard **OpenID Connect (OIDC)** and served behind any modern reverse proxy (Traefik, Caddy, Nginx).

Built with **Go**, **SQLite (WAL Mode)**, and **React (TypeScript + Tailwind CSS)** embedded directly into a single production binary via `go:embed`.

---

## Features

- **Authenticated DoH Access**: Only users authenticated and authorized by your OpenID Connect provider can register devices and query DoH.
- **Provider Agnostic (OIDC Standard)**: Fully compliant with any standard OpenID Connect provider via discovery (`/.well-known/openid-configuration`):
  - Zitadel
  - Authentik
  - Keycloak
  - Authelia
  - Google Identity
  - Okta / Auth0
- **Reverse Proxy Agnostic**: Works behind any reverse proxy terminating TLS (Traefik, Caddy, Nginx, Envoy, Cloudflare).
- **Client Path Isolation**: Every device is provisioned a unique token (e.g. `https://dns.example.com/dns-query/u-8f3a12b4`) registered directly in AdGuard Home's client list and access control list.
- **Performance-Safe Filtering Controls**:
  - **Ad & Tracker Blocking**: Instant toggle without reloading global filter engines.
  - **SafeSearch**: Enforced per-device (Google, Bing, YouTube).
  - **Parental Controls**: One-click toggle for adult content filtering.
  - **Blocked Services**: Pre-compiled blocking for TikTok, YouTube, Steam, Discord, Reddit, etc.
  - **Exact Domain Allow/Block Overrides**: Capped at 50 domains per device to protect RAM and DNS query speed (disallows dangerous ReDoS patterns).
- **1-Click Apple Setup**: Generates and downloads native `.mobileconfig` profiles for iOS and macOS without installing third-party apps.
- **Single Portable Binary**: The compiled React SPA is embedded into the Go binary via `go:embed`.

---

## Architecture

```text
User Device
    │
    ├── (HTTPS Web UI) ────────> Reverse Proxy ──> AdGuard Portal (Go + SQLite + React)
    │                                                   │ (Syncs /control/clients/*)
    └── (DoH /dns-query/<token>) > Reverse Proxy ──> AdGuard Home (Engine)
```

---

## Configuration (`.env`)

| Variable | Description | Default |
| :--- | :--- | :--- |
| `PORT` | HTTP port for the web app | `8080` |
| `DB_PATH` | Path to SQLite database file | `data/portal.db` |
| `DOH_DOMAIN` | Public domain serving DoH | `dns.example.com` |
| `AGH_URL` | Internal URL to AdGuard Home | `http://127.0.0.1:3000` |
| `AGH_USER` | AdGuard Home Admin Username | `""` |
| `AGH_PASSWORD` | AdGuard Home Admin Password | `""` |
| `OIDC_ISSUER` | OIDC Provider Issuer URL | `https://auth.example.com` |
| `OIDC_CLIENT_ID` | OIDC Application Client ID | `""` |
| `OIDC_SCOPES` | OIDC Scopes requested | `openid profile email` |
| `OIDC_PROVIDER_NAME` | Display name shown on sign-in button | `Single Sign-On` |
| `MAX_DEVICES_PER_USER` | Quota of devices per user | `10` |
| `DEV_MODE` | Enable developer mock login for offline testing | `false` |

---

## Setting up Your OIDC Provider

In your OIDC provider (Authentik, Zitadel, Keycloak, Authelia, Google, etc.):

1. Create a **Single Page Application (SPA)** or **Web Application** client.
2. Select **Authorization Code with PKCE** (`response_type=code`).
3. Set **Redirect URIs**:
   - Production: `https://portal.example.com/`
   - Local dev: `http://localhost:5173/` and `http://localhost:8080/`
4. Set **Post Logout Redirect URIs**: `https://portal.example.com/`
5. Configure your provider's authorization policies or invite-only rules to restrict which users can log in.

---

## Deployment with Docker

A production-ready Docker Compose example using Traefik is provided in `docker-compose.example.yml`:

```bash
# 1. Copy environment template
cp .env.example .env

# 2. Build and start containers
docker compose -f docker-compose.example.yml up -d --build
```

### Alternative: Caddyfile Example
```caddy
portal.example.com {
    reverse_proxy localhost:8080
}

dns.example.com {
    reverse_proxy /dns-query/* localhost:3000
}
```

---

## Local Development

### Prerequisites
- Go 1.22+
- Node.js 20+ and `pnpm`

### Running Locally
```bash
# Terminal 1: Run Go Backend (with DEV_MODE enabled)
export DEV_MODE=true
export DOH_DOMAIN=localhost
go run ./cmd/server

# Terminal 2: Run React Frontend with Hot Reload
cd frontend
pnpm install
pnpm dev
```

Visit `http://localhost:5173` to test the UI. The Vite dev server proxies `/api` calls directly to the Go backend.

---

## License
MIT
