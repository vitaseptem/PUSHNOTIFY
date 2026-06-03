'use client';

import Image from 'next/image';
import Link from 'next/link';
import { useState } from 'react';

type Lang = 'en' | 'pt';

function Code({ children }: { children: string }) {
  return (
    <pre className="my-3 overflow-x-auto rounded-xl border border-border bg-[#0d0d14] p-4 font-mono text-xs leading-relaxed text-foreground">
      <code>{children}</code>
    </pre>
  );
}

const NAV: Array<{ id: string; en: string; pt: string }> = [
  { id: 'intro', en: 'Introduction', pt: 'Introdução' },
  { id: 'quickstart', en: 'Quick start', pt: 'Início rápido' },
  { id: 'auth', en: 'Authentication', pt: 'Autenticação' },
  { id: 'send', en: 'Sending notifications', pt: 'Enviando notificações' },
  { id: 'channels', en: 'Channels', pt: 'Canais' },
  { id: 'subscribers', en: 'Subscribers', pt: 'Subscribers' },
  { id: 'templates', en: 'Templates', pt: 'Templates' },
  { id: 'realtime', en: 'Realtime (WebSocket)', pt: 'Tempo real (WebSocket)' },
  { id: 'webpush', en: 'Web Push', pt: 'Web Push' },
  { id: 'webhooks', en: 'Webhooks', pt: 'Webhooks' },
  { id: 'ratelimits', en: 'Rate limits', pt: 'Limites de taxa' },
  { id: 'errors', en: 'Errors', pt: 'Erros' },
  { id: 'plans', en: 'Plans & billing', pt: 'Planos e cobrança' },
];

export default function DocsPage() {
  const [lang, setLang] = useState<Lang>('en');
  const t = (en: string, pt: string) => (lang === 'en' ? en : pt);

  return (
    <div className="min-h-screen bg-background text-foreground">
      <header className="sticky top-0 z-10 flex items-center justify-between border-b border-border bg-surface/90 px-6 py-3 backdrop-blur">
        <Link href="/dashboard" className="flex items-center gap-2">
          <Image src="/logo.png" alt="PushNotify" width={150} height={32} className="h-8 w-auto" />
        </Link>
        <div className="flex items-center gap-1 rounded-lg border border-border p-1 text-sm">
          <button
            onClick={() => setLang('en')}
            className={`rounded px-2 py-0.5 ${lang === 'en' ? 'bg-primary text-white' : 'text-muted'}`}
          >
            🇺🇸 EN
          </button>
          <button
            onClick={() => setLang('pt')}
            className={`rounded px-2 py-0.5 ${lang === 'pt' ? 'bg-primary text-white' : 'text-muted'}`}
          >
            🇧🇷 PT
          </button>
        </div>
      </header>

      <div className="mx-auto flex max-w-6xl gap-8 px-6 py-8">
        <nav className="sticky top-20 hidden h-fit w-52 shrink-0 space-y-1 text-sm lg:block">
          {NAV.map((n) => (
            <a key={n.id} href={`#${n.id}`} className="block rounded px-2 py-1 text-muted hover:bg-white/5 hover:text-foreground">
              {t(n.en, n.pt)}
            </a>
          ))}
        </nav>

        <main className="prose-invert min-w-0 flex-1 space-y-12">
          <Section id="intro" title={t('Introduction', 'Introdução')}>
            <p className="text-muted">
              {t(
                'PushNotify is real-time, multi-channel notification infrastructure. Send notifications over WebSocket, Web Push, Email, WhatsApp and SMS through a single REST API or the TypeScript SDK.',
                'PushNotify é uma infraestrutura de notificações multicanal em tempo real. Envie notificações por WebSocket, Web Push, E-mail, WhatsApp e SMS por uma única API REST ou pelo SDK TypeScript.',
              )}
            </p>
            <p className="text-muted">
              {t('The API base URL of a self-hosted instance is, by default:', 'A URL base da API de uma instância self-hosted é, por padrão:')}
            </p>
            <Code>{`https://api.pushnotify.dev      # production
http://localhost:8080          # local (docker compose)`}</Code>
          </Section>

          <Section id="quickstart" title={t('Quick start', 'Início rápido')}>
            <p className="text-muted">{t('Run the whole stack locally:', 'Suba toda a stack localmente:')}</p>
            <Code>{`git clone https://github.com/vitaseptem/PUSHNOTIFY.git && cd PUSHNOTIFY
cp .env.example .env
make dev          # API :8080 · dashboard :3000 · nginx :80
make seed         # demo@pushnotify.dev / demo1234 + an API key`}</Code>
            <p className="text-muted">{t('Install the SDK:', 'Instale o SDK:')}</p>
            <Code>{`npm install @astrazstudio/pushnotify-sdk`}</Code>
          </Section>

          <Section id="auth" title={t('Authentication', 'Autenticação')}>
            <p className="text-muted">
              {t(
                'Server-to-server requests use an API key in the X-API-Key header. The dashboard uses a JWT (Authorization: Bearer). Create API keys in Settings → API Keys.',
                'Requisições servidor-a-servidor usam uma API key no header X-API-Key. O dashboard usa um JWT (Authorization: Bearer). Crie API keys em Settings → API Keys.',
              )}
            </p>
            <Code>{`# API key (server-side)
curl https://api.pushnotify.dev/api/v1/notifications \\
  -H "X-API-Key: pk_live_xxx"

# JWT (dashboard / user)
curl https://api.pushnotify.dev/api/v1/auth/me \\
  -H "Authorization: Bearer eyJhbGciOi..."`}</Code>
            <p className="text-muted">
              {t(
                'Register and log in to obtain a JWT:',
                'Registre-se e faça login para obter um JWT:',
              )}
            </p>
            <Code>{`POST /api/v1/auth/register   { "email", "full_name", "password", "workspace_name" }
POST /api/v1/auth/login      { "email", "password" }   ->   { "token", "user", "workspace" }`}</Code>
          </Section>

          <Section id="send" title={t('Sending notifications', 'Enviando notificações')}>
            <p className="text-muted">{t('Single send via REST:', 'Envio único via REST:')}</p>
            <Code>{`POST /api/v1/notifications/send
X-API-Key: pk_live_xxx
Content-Type: application/json

{
  "subscriber_id": "user_123",
  "channels": ["websocket", "email"],
  "subject": "New order!",
  "body": "Your order #1234 was confirmed.",
  "priority": 2,
  "metadata": { "url": "https://app.com/orders/1234" }
}`}</Code>
            <p className="text-muted">{t('With the SDK:', 'Com o SDK:')}</p>
            <Code>{`import { PushNotifyClient } from '@astrazstudio/pushnotify-sdk';

const client = new PushNotifyClient({ apiKey: 'pk_live_xxx' });

await client.send('user_123', {
  channels: ['websocket', 'email'],
  subject: 'New order!',
  body: 'Your order #1234 was confirmed.',
});`}</Code>
            <p className="text-muted">{t('Bulk send (batches of up to 100):', 'Envio em lote (batches de até 100):')}</p>
            <Code>{`POST /api/v1/notifications/send-bulk
{ "notifications": [ { "subscriber_id": "u1", "body": "Hi", "channels": ["email"] }, ... ] }`}</Code>
          </Section>

          <Section id="channels" title={t('Channels', 'Canais')}>
            <p className="text-muted">
              {t(
                'WebSocket and Web Push are managed by PushNotify. Email/WhatsApp/SMS are configured via server environment variables (SMTP_*, EVOLUTION_API_*, TWILIO_*). WebSocket is best-effort: an offline subscriber is not an error.',
                'WebSocket e Web Push são gerenciados pelo PushNotify. Email/WhatsApp/SMS são configurados por variáveis de ambiente do servidor (SMTP_*, EVOLUTION_API_*, TWILIO_*). WebSocket é best-effort: um subscriber offline não é erro.',
              )}
            </p>
            <Code>{`websocket  - realtime, browser/app connections
webpush    - browser push via VAPID
email      - SMTP
whatsapp   - Evolution API / Z-API
sms        - Twilio (pluggable)`}</Code>
          </Section>

          <Section id="subscribers" title="Subscribers">
            <p className="text-muted">
              {t(
                'A subscriber is an end user identified by your own external_id. Create or update one before sending:',
                'Um subscriber é um usuário final identificado pelo seu próprio external_id. Crie ou atualize antes de enviar:',
              )}
            </p>
            <Code>{`POST /api/v1/subscribers
{ "external_id": "user_123", "email": "ana@ex.com", "whatsapp": "+5511999999999" }

// SDK
await client.upsertSubscriber('user_123', { email: 'ana@ex.com' });`}</Code>
          </Section>

          <Section id="templates" title="Templates">
            <p className="text-muted">
              {t(
                'Templates store reusable per-channel bodies with {{variables}}. Send by slug:',
                'Templates guardam corpos reutilizáveis por canal com {{variaveis}}. Envie pelo slug:',
              )}
            </p>
            <Code>{`await client.sendTemplate('user_123', 'order-confirmed', {
  name: 'Ana', order_id: '1234'
});`}</Code>
          </Section>

          <Section id="realtime" title={t('Realtime (WebSocket)', 'Tempo real (WebSocket)')}>
            <p className="text-muted">
              {t(
                'Issue a short-lived subscriber token server-side, then connect from the browser:',
                'Emita um token de subscriber no servidor e conecte do navegador:',
              )}
            </p>
            <Code>{`// server: mint a token
const token = await client.issueWebSocketToken('user_123');

// browser
const conn = client.connect('workspace_id', 'user_123', token);
conn.onMessage((n) => console.log(n.subject, n.body));`}</Code>
            <p className="text-muted">{t('Raw endpoint:', 'Endpoint cru:')}</p>
            <Code>{`GET wss://api.pushnotify.dev/ws/{workspaceID}/{subscriberID}?token=JWT`}</Code>
          </Section>

          <Section id="webpush" title="Web Push">
            <Code>{`import { subscribeWebPush } from '@astrazstudio/pushnotify-sdk';

const sub = await subscribeWebPush(vapidPublicKey, '/sw.js');
await client.registerWebPush('user_123', sub);`}</Code>
            <p className="text-muted">
              {t(
                'The VAPID public key is shown in Channels. Expired subscriptions (HTTP 410) are auto-deactivated.',
                'A chave pública VAPID aparece em Channels. Subscriptions expiradas (HTTP 410) são desativadas automaticamente.',
              )}
            </p>
          </Section>

          <Section id="webhooks" title="Webhooks">
            <p className="text-muted">
              {t(
                'Register a URL to receive delivery status events. Each request is signed with HMAC-SHA256 over the raw body using your webhook secret.',
                'Registre uma URL para receber eventos de status de entrega. Cada requisição é assinada com HMAC-SHA256 sobre o corpo cru usando o secret do webhook.',
              )}
            </p>
            <Code>{`Events: delivery.sent · delivery.delivered · delivery.failed
Header: X-PushNotify-Signature: sha256=<hex>`}</Code>
            <p className="text-muted">{t('Verify in Node.js:', 'Verifique no Node.js:')}</p>
            <Code>{`import crypto from 'crypto';

function verify(rawBody, signature, secret) {
  const expected = 'sha256=' + crypto.createHmac('sha256', secret)
    .update(rawBody).digest('hex');
  return crypto.timingSafeEqual(Buffer.from(signature), Buffer.from(expected));
}`}</Code>
          </Section>

          <Section id="ratelimits" title={t('Rate limits', 'Limites de taxa')}>
            <p className="text-muted">
              {t(
                'Public routes: 100 req/min per IP. Authenticated routes: 1000 req/min per workspace. Responses include X-RateLimit-Limit and X-RateLimit-Remaining; exceeding returns HTTP 429.',
                'Rotas públicas: 100 req/min por IP. Rotas autenticadas: 1000 req/min por workspace. As respostas incluem X-RateLimit-Limit e X-RateLimit-Remaining; ao exceder retorna HTTP 429.',
              )}
            </p>
          </Section>

          <Section id="errors" title={t('Errors', 'Erros')}>
            <p className="text-muted">
              {t('Errors are JSON with an "error" field and a standard HTTP status:', 'Erros são JSON com um campo "error" e status HTTP padrão:')}
            </p>
            <Code>{`400 invalid request    401 unauthorized      404 not found
409 conflict           429 rate limited      500 server error

{ "error": "subscriber \\"user_123\\" not found" }`}</Code>
            <p className="text-muted">
              {t(
                'The SDK throws PushNotifyError (with status) and retries 5xx automatically (3 attempts).',
                'O SDK lança PushNotifyError (com status) e tenta novamente em 5xx automaticamente (3 tentativas).',
              )}
            </p>
          </Section>

          <Section id="plans" title={t('Plans & billing', 'Planos e cobrança')}>
            <Code>{`Free        10,000 notifications/mo · 3 channels · free
Pro        500,000 notifications/mo · all channels · $29/mo
Enterprise unlimited · all + SLA · contact`}</Code>
            <p className="text-muted">
              {t(
                'Upgrade from Settings → Plan (Stripe Checkout). Usage is tracked per workspace and enforced on send.',
                'Faça upgrade em Settings → Plan (Stripe Checkout). O uso é contado por workspace e validado no envio.',
              )}
            </p>
          </Section>

          <footer className="border-t border-border pt-6 text-sm text-muted">
            {t('Built with ❤️ by Astraz Studio', 'Feito com ❤️ pela Astraz Studio')} ·{' '}
            <Link href="/dashboard" className="text-[#E98BA0] hover:underline">
              {t('Back to dashboard', 'Voltar ao dashboard')}
            </Link>
          </footer>
        </main>
      </div>
    </div>
  );
}

function Section({ id, title, children }: { id: string; title: string; children: React.ReactNode }) {
  return (
    <section id={id} className="scroll-mt-20">
      <h2 className="mb-3 text-xl font-semibold">{title}</h2>
      {children}
    </section>
  );
}
