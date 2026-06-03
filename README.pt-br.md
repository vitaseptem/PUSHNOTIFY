<p align="center">
  <img src="./assets/logo.png" alt="PushNotify" width="320" />
</p>

<h1 align="center">PushNotify 🔔</h1>

<p align="center">
  <strong>Infraestrutura de notificações multicanal em tempo real para desenvolvedores.</strong><br/>
  Criado pela <a href="https://github.com/vitaseptem/PUSHNOTIFY">Astraz Studio</a>.
</p>

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white" />
  <img alt="License" src="https://img.shields.io/badge/License-MIT-7B1C2E" />
  <img alt="Docker" src="https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white" />
  <img alt="Status" src="https://img.shields.io/badge/status-v1.0.0-22C55E" />
</p>

<p align="center">
  <a href="./README.md">🇺🇸 English</a> · <strong>🇧🇷 Português</strong>
</p>

---

## O que é

O PushNotify é um backend de **notificações como serviço** (NaaS) que você pode
auto-hospedar ou revender. Qualquer desenvolvedor o integra em minutos via API
REST ou pelo SDK em TypeScript e passa a entregar notificações por WebSocket,
Web Push, E-mail, WhatsApp e SMS — com retentativas, fila de mensagens mortas
(DLQ), templates, workspaces multi-tenant, webhooks e um dashboard em tempo real.

Pense em *Twilio Notify + Novu + Knock* — independente, auto-hospedável e com
modelo pay-per-use.

## Canais suportados

| Canal      | Protocolo                            | Status |
|------------|--------------------------------------|:------:|
| WebSocket  | Gorilla WS + fan-out via Redis pub/sub | ✅ |
| Web Push   | VAPID (`webpush-go`)                 | ✅ |
| E-mail     | SMTP (HTML, template dark)           | ✅ |
| WhatsApp   | Evolution API / Z-API                | ✅ |
| SMS        | Twilio (plugável)                    | ✅ |
| Telegram   | —                                    | 🚧 |

## Início rápido (3 comandos)

```bash
git clone https://github.com/vitaseptem/PUSHNOTIFY.git && cd PUSHNOTIFY
cp .env.example .env
make dev
```

Depois acesse:

- Dashboard → http://localhost:3000
- API → http://localhost:8080 (health: `/health`)
- Via nginx → http://localhost

Popule uma conta de demonstração (login `demo@pushnotify.dev` / `demo1234`, com uma API key):

```bash
make seed
```

## Integre em 60 segundos

```bash
npm install @astrazstudio/pushnotify-sdk
```

```ts
import { PushNotifyClient } from '@astrazstudio/pushnotify-sdk';

const client = new PushNotifyClient({ apiKey: 'pk_...' });

await client.send('user_123', {
  channels: ['websocket', 'email'],
  subject: 'Novo pedido!',
  body: 'Seu pedido #1234 foi confirmado.',
});
```

## API (REST)

Todos os endpoints ficam sob `/api/v1`. A autenticação é por JWT
(`Authorization: Bearer`) para o dashboard, ou por API key
(`X-API-Key: pk_...`) para integrações servidor-a-servidor.

| Método | Endpoint | Auth | Descrição |
|--------|----------|------|-----------|
| POST | `/auth/register` | público | Cria conta + workspace |
| POST | `/auth/login` | público | Obtém um JWT |
| GET  | `/auth/me` | JWT | Usuário + workspace atuais |
| GET/PUT | `/workspace` | JWT | Lê / atualiza workspace |
| POST/GET/DELETE | `/workspace/api-keys` | JWT | Gerencia API keys |
| GET | `/workspace/usage` | JWT | Uso do plano |
| POST | `/notifications/send` | API key / JWT | Envia uma notificação |
| POST | `/notifications/send-bulk` | API key / JWT | Envio em lote |
| GET | `/notifications` | API key / JWT | Lista notificações |
| GET | `/notifications/{id}` | API key / JWT | Notificação + entregas |
| GET/POST/PUT/DELETE | `/templates` | JWT | Gerencia templates |
| POST | `/templates/{id}/test` | JWT | Renderiza um template |
| GET/POST/PUT/DELETE | `/subscribers` | API key / JWT | Gerencia subscribers |
| POST | `/subscribers/{externalId}/web-push` | API key / JWT | Registra Web Push |
| GET/POST/DELETE | `/webhooks` | JWT | Gerencia webhooks de status |
| GET | `/dashboard/overview` | JWT | Visão geral (com cache) |
| GET | `/dashboard/analytics?period=7d` | JWT | Analytics em série temporal |
| GET | `/dashboard/live` | JWT | Métricas ao vivo (SSE) |
| GET | `/ws/{workspaceID}/{subscriberID}?token=…` | JWT do subscriber | Stream WebSocket |

## Arquitetura

```
  App ──▶ API REST ──▶ Fila (Redis) ──▶ Workers ──▶ Canais ──▶ Subscribers
                │                           │
                ▼                           ▼
           PostgreSQL                Retry + Backoff
                │                           │
                ▼                           ▼
        WebSocket Hub ◀── Redis pub/sub ── Dead Letter Queue
                │
                ▼
           Navegadores
```

- **Filas por prioridade** (alta / normal / baixa) em listas do Redis.
- **Retentativa** com backoff exponencial + jitter (limite de 1h); jobs esgotados vão para a **fila de mensagens mortas**.
- **WebSocket Hub** faz fan-out entre nós da API via Redis pub/sub, com presença no Redis.
- **Desligamento gracioso**: ao receber `SIGTERM`, o servidor HTTP para de aceitar e o worker drena os jobs em andamento.

## Modelo de preços

| Plano | Volume | Canais | Preço |
|-------|--------|--------|-------|
| Free | 10.000 notif/mês | 3 canais | grátis |
| Pro | 500.000 notif/mês | todos os canais | $29/mês |
| Enterprise | ilimitado | todos + SLA | contato |

## Auto-hospedagem

Tudo é distribuído como imagens Docker. Em uma única VPS:

```bash
cp .env.example .env   # defina um JWT_SECRET forte e o NEXT_PUBLIC_API_URL
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build
```

Coloque o Cloudflare (SSL/CDN grátis) na frente do nginx. Divisão gerenciada
recomendada: servidor Go + workers no Railway, Postgres no Railway, Redis no
Upstash, dashboard na Vercel, domínio em `pushnotify.dev`.

## Estrutura do repositório

```
server/      Backend em Go (API + workers + fila + canais + hub)
dashboard/   Dashboard Next.js 14 (TypeScript, Tailwind, Recharts)
sdk/         SDK TypeScript publicável (zero deps, ESM + CJS + d.ts)
nginx/       Proxy reverso (API + WS + dashboard)
```

## Desenvolvimento

```bash
make test        # go test ./...
make sdk         # build do SDK TypeScript
make dashboard   # build do dashboard
make server      # build do binário Go
```

## Roadmap

- [ ] SDK Mobile (React Native)
- [ ] Canal Telegram
- [ ] Regras de segmentação
- [ ] Teste A/B de templates
- [ ] Analytics avançado

## Licença

MIT — veja [LICENSE](./LICENSE). Feito com ❤️ pela **Astraz Studio** · 2026
