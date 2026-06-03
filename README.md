<p align="center">
  <img src="./assets/logo.png" alt="PushNotify" width="320" />
</p>

<h1 align="center">PushNotify 🔔</h1>

<p align="center">
  <strong>Real-time, multi-channel notification infrastructure for developers.</strong><br/>
  Built by <a href="https://github.com/vitaseptem/PUSHNOTIFY">Astraz Studio</a>.
</p>

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white" />
  <img alt="License" src="https://img.shields.io/badge/License-MIT-7B1C2E" />
  <img alt="Docker" src="https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white" />
  <img alt="Status" src="https://img.shields.io/badge/status-v1.0.0-22C55E" />
</p>

---

## What is it

PushNotify is a **notifications-as-a-service** (NaaS) backend you can self-host or
sell. Any developer integrates it in minutes via the REST API or the TypeScript
SDK and starts delivering notifications across WebSocket, Web Push, Email,
WhatsApp and SMS — with retries, dead-letter queues, templates, multi-tenant
workspaces, webhooks and a real-time dashboard.

Think *Twilio Notify + Novu + Knock* — independent, self-hostable, pay-per-use.

## Supported channels

| Channel    | Protocol                         | Status |
|------------|----------------------------------|:------:|
| WebSocket  | Gorilla WS + Redis pub/sub fan-out | ✅ |
| Web Push   | VAPID (`webpush-go`)             | ✅ |
| Email      | SMTP (HTML, dark template)       | ✅ |
| WhatsApp   | Evolution API / Z-API            | ✅ |
| SMS        | Twilio (pluggable)               | ✅ |
| Telegram   | —                                | 🚧 |

## Quick start (3 commands)

```bash
git clone https://github.com/vitaseptem/PUSHNOTIFY.git && cd PUSHNOTIFY
cp .env.example .env
make dev
```

Then open:

- Dashboard → http://localhost:3000
- API → http://localhost:8080 (health: `/health`)
- Through nginx → http://localhost

Seed a demo account (login `demo@pushnotify.dev` / `demo1234`, plus an API key):

```bash
make seed
```

## Integrate in 60 seconds

```bash
npm install @astrazstudio/pushnotify-sdk
```

```ts
import { PushNotifyClient } from '@astrazstudio/pushnotify-sdk';

const client = new PushNotifyClient({ apiKey: 'pk_...' });

await client.send('user_123', {
  channels: ['websocket', 'email'],
  subject: 'New order!',
  body: 'Your order #1234 has been confirmed.',
});
```

## API (REST)

All endpoints are versioned under `/api/v1`. Auth is a JWT (`Authorization: Bearer`)
for the dashboard, or an API key (`X-API-Key: pk_...`) for server-to-server.

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/auth/register` | public | Create account + workspace |
| POST | `/auth/login` | public | Get a JWT |
| GET  | `/auth/me` | JWT | Current user + workspace |
| GET/PUT | `/workspace` | JWT | Read / update workspace |
| POST/GET/DELETE | `/workspace/api-keys` | JWT | Manage API keys |
| GET | `/workspace/usage` | JWT | Plan usage |
| POST | `/notifications/send` | API key / JWT | Send a notification |
| POST | `/notifications/send-bulk` | API key / JWT | Bulk send |
| GET | `/notifications` | API key / JWT | List notifications |
| GET | `/notifications/{id}` | API key / JWT | Notification + deliveries |
| GET/POST/PUT/DELETE | `/templates` | JWT | Manage templates |
| POST | `/templates/{id}/test` | JWT | Render a template |
| GET/POST/PUT/DELETE | `/subscribers` | API key / JWT | Manage subscribers |
| POST | `/subscribers/{externalId}/web-push` | API key / JWT | Register Web Push |
| GET/POST/DELETE | `/webhooks` | JWT | Manage status webhooks |
| GET | `/dashboard/overview` | JWT | Cached metrics overview |
| GET | `/dashboard/analytics?period=7d` | JWT | Time-series analytics |
| GET | `/dashboard/live` | JWT | SSE live metrics |
| GET | `/ws/{workspaceID}/{subscriberID}?token=…` | subscriber JWT | WebSocket stream |

## Architecture

```
  App ──▶ REST API ──▶ Queue (Redis) ──▶ Workers ──▶ Channels ──▶ Subscribers
                │                            │
                ▼                            ▼
           PostgreSQL                 Retry + Backoff
                │                            │
                ▼                            ▼
        WebSocket Hub ◀── Redis pub/sub ── Dead Letter Queue
                │
                ▼
            Browsers
```

- **Priority queues** (high / default / low) on Redis lists.
- **Retry** with exponential backoff + jitter (capped at 1h); exhausted jobs go to a **dead-letter queue**.
- **WebSocket Hub** fans out across API nodes via Redis pub/sub, with Redis-backed presence.
- **Graceful shutdown**: on `SIGTERM` the HTTP server stops accepting and the worker drains in-flight jobs.

## Pricing model

| Plan | Volume | Channels | Price |
|------|--------|----------|-------|
| Free | 10,000 notif/mo | 3 channels | free |
| Pro | 500,000 notif/mo | all channels | $29/mo |
| Enterprise | unlimited | all + SLA | contact |

## Self-hosting

Everything ships as Docker images. On a single VPS:

```bash
cp .env.example .env   # set a strong JWT_SECRET and NEXT_PUBLIC_API_URL
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

Put Cloudflare (free SSL/CDN) in front of nginx. Recommended managed split:
Go server + workers on Railway, Postgres on Railway, Redis on Upstash,
dashboard on Vercel, domain on `pushnotify.dev`.

## Repository layout

```
server/      Go backend (API + workers + queue + channels + hub)
dashboard/   Next.js 14 dashboard (TypeScript, Tailwind, Recharts)
sdk/         Publishable TypeScript SDK (zero deps, ESM + CJS + d.ts)
nginx/       Reverse proxy (API + WS + dashboard)
```

## Development

```bash
make test        # go test ./...
make sdk         # build the TypeScript SDK
make dashboard   # build the dashboard
make server      # build the Go binary
```

## Roadmap

- [ ] SDK Mobile (React Native)
- [ ] Telegram channel
- [ ] Segmentation rules
- [ ] A/B testing of templates
- [ ] Advanced analytics

## License

MIT · Built with ❤️ by **Astraz Studio** · 2026
