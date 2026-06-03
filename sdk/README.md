# @astrazstudio/pushnotify-sdk

Official TypeScript SDK for **[PushNotify](https://github.com/vitaseptem/PUSHNOTIFY)** — real-time, multi-channel notification infrastructure by Astraz Studio.

Zero runtime dependencies. Works in Node.js (≥18) and the browser. Ships ESM, CJS and type definitions.

## Install

```bash
npm install @astrazstudio/pushnotify-sdk
```

## Send a notification (server-side)

```ts
import { PushNotifyClient } from '@astrazstudio/pushnotify-sdk';

const client = new PushNotifyClient({
  apiKey: 'pk_your_workspace_key',
  baseUrl: 'https://api.pushnotify.dev',
});

await client.send('user_123', {
  channels: ['websocket', 'email'],
  subject: 'New order!',
  body: 'Your order #1234 has been confirmed.',
  priority: 2,
});
```

## Send with a template

```ts
await client.sendTemplate('user_123', 'order-confirmed', {
  order_id: '1234',
  name: 'Ana',
});
```

## Bulk send

```ts
await client.sendBulk([
  { subscriber_id: 'user_1', body: 'Hello 1', channels: ['email'] },
  { subscriber_id: 'user_2', body: 'Hello 2', channels: ['websocket'] },
]);
```

## Manage subscribers

```ts
await client.upsertSubscriber('user_123', {
  email: 'ana@example.com',
  whatsapp: '+5511999999999',
  metadata: { plan: 'pro' },
});
```

## Receive in real time (browser)

```ts
const token = await client.issueWebSocketToken('user_123');
const conn = client.connect('workspace_id', 'user_123', token);

conn.onMessage((notification) => {
  console.log('received', notification.subject, notification.body);
});

// later
client.disconnect();
```

## Web Push (browser)

```ts
import { subscribeWebPush } from '@astrazstudio/pushnotify-sdk';

const subscription = await subscribeWebPush(vapidPublicKey, '/sw.js');
await client.registerWebPush('user_123', subscription);
```

## Error handling

All failures throw `PushNotifyError` (with `status` and `body`). Timeouts throw
`PushNotifyTimeoutError`. Requests retry automatically on `5xx` responses
(default 3 attempts).

```ts
import { PushNotifyError } from '@astrazstudio/pushnotify-sdk';

try {
  await client.send('user_123', { body: 'hi' });
} catch (err) {
  if (err instanceof PushNotifyError) {
    console.error(err.status, err.message);
  }
}
```

## License

MIT · Built with ❤️ by Astraz Studio
