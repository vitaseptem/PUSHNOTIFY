'use client';

import { useEffect, useState } from 'react';
import { Badge, Button, Card, EmptyState, ErrorState, Input, Modal, Skeleton } from '@/components/ui';
import { api } from '@/lib/api';
import { timeAgo } from '@/lib/utils';
import type { Webhook } from '@/types';

const EVENTS = ['delivery.sent', 'delivery.delivered', 'delivery.failed'];

export default function WebhooksPage() {
  const [hooks, setHooks] = useState<Webhook[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [open, setOpen] = useState(false);
  const [newSecret, setNewSecret] = useState<string | null>(null);

  function load() {
    setLoading(true);
    api<{ data: Webhook[] | null }>('/api/v1/webhooks')
      .then((res) => setHooks(res.data ?? []))
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }

  useEffect(load, []);

  async function remove(id: string) {
    await api(`/api/v1/webhooks/${id}`, { method: 'DELETE' });
    load();
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button onClick={() => setOpen(true)}>Add webhook</Button>
      </div>
      {error && <ErrorState message={error} />}
      {newSecret && (
        <Card className="border-success/30 bg-success/10">
          <p className="text-sm text-success">Webhook created. Save this signing secret — it is shown only once:</p>
          <p className="mt-2 break-all font-mono text-xs">{newSecret}</p>
        </Card>
      )}
      {loading ? (
        <Skeleton className="h-24" />
      ) : hooks.length === 0 ? (
        <EmptyState title="No webhooks configured" hint="Receive HMAC-signed delivery status events." />
      ) : (
        <div className="space-y-3">
          {hooks.map((h) => (
            <Card key={h.id} className="flex items-center justify-between">
              <div>
                <p className="break-all font-mono text-sm">{h.url}</p>
                <div className="mt-2 flex flex-wrap gap-1">
                  {h.events.map((e) => (
                    <Badge key={e} color="primary">
                      {e}
                    </Badge>
                  ))}
                </div>
                <p className="mt-2 text-xs text-muted">Created {timeAgo(h.created_at)}</p>
              </div>
              <Button variant="danger" onClick={() => remove(h.id)}>
                Delete
              </Button>
            </Card>
          ))}
        </div>
      )}
      <AddWebhookModal
        open={open}
        onClose={() => setOpen(false)}
        onCreated={(secret) => {
          setOpen(false);
          setNewSecret(secret);
          load();
        }}
      />
    </div>
  );
}

function AddWebhookModal({
  open,
  onClose,
  onCreated,
}: {
  open: boolean;
  onClose: () => void;
  onCreated: (secret: string) => void;
}) {
  const [url, setUrl] = useState('');
  const [events, setEvents] = useState<string[]>(['delivery.delivered', 'delivery.failed']);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  function toggle(e: string) {
    setEvents((prev) => (prev.includes(e) ? prev.filter((x) => x !== e) : [...prev, e]));
  }

  async function submit() {
    setLoading(true);
    setError(null);
    try {
      const res = await api<{ secret: string }>('/api/v1/webhooks', {
        method: 'POST',
        body: { url, events },
      });
      onCreated(res.secret);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Add webhook">
      <div className="space-y-4">
        {error && <ErrorState message={error} />}
        <Input placeholder="https://your-app.com/webhooks/pushnotify" value={url} onChange={(e) => setUrl(e.target.value)} />
        <div className="flex flex-wrap gap-2">
          {EVENTS.map((e) => (
            <button
              key={e}
              type="button"
              onClick={() => toggle(e)}
              className={`rounded-md px-3 py-1 text-xs ${
                events.includes(e) ? 'bg-primary text-white' : 'bg-white/5 text-muted'
              }`}
            >
              {e}
            </button>
          ))}
        </div>
        <Button onClick={submit} loading={loading} disabled={!url || events.length === 0} className="w-full">
          Create webhook
        </Button>
      </div>
    </Modal>
  );
}
