'use client';

import { useEffect, useState } from 'react';
import { Badge, Button, Card, ErrorState, Skeleton } from '@/components/ui';
import { api } from '@/lib/api';
import type { Workspace } from '@/types';

export default function ChannelsPage() {
  const [ws, setWs] = useState<Workspace | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [regenerating, setRegenerating] = useState(false);

  function load() {
    api<Workspace>('/api/v1/workspace')
      .then(setWs)
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }

  useEffect(load, []);

  async function regenerate() {
    setRegenerating(true);
    try {
      await api('/api/v1/workspace/vapid/regenerate', { method: 'POST' });
      load();
    } finally {
      setRegenerating(false);
    }
  }

  if (loading) return <Skeleton className="h-40" />;
  if (error) return <ErrorState message={error} />;

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <ChannelCard name="Email (SMTP)" desc="Configure via SMTP_* environment variables on the server." badge="env" />
      <ChannelCard name="WhatsApp" desc="Evolution API / Z-API. Set EVOLUTION_API_URL and EVOLUTION_API_KEY." badge="env" />
      <ChannelCard name="SMS" desc="Twilio. Set TWILIO_SID, TWILIO_TOKEN and TWILIO_FROM." badge="env" />
      <ChannelCard name="WebSocket" desc="Real-time delivery is always enabled for connected subscribers." badge="active" />

      <Card className="lg:col-span-2">
        <div className="flex items-center justify-between">
          <div>
            <p className="font-medium">Web Push (VAPID)</p>
            <p className="text-sm text-muted">Public key used by browsers to subscribe.</p>
          </div>
          <Button variant="secondary" onClick={regenerate} loading={regenerating}>
            Regenerate keys
          </Button>
        </div>
        <p className="mt-3 break-all rounded-lg border border-border bg-background p-3 font-mono text-xs">
          {ws?.vapid_public_key || '—'}
        </p>
      </Card>
    </div>
  );
}

function ChannelCard({ name, desc, badge }: { name: string; desc: string; badge: string }) {
  return (
    <Card>
      <div className="flex items-center justify-between">
        <p className="font-medium">{name}</p>
        <Badge color={badge === 'active' ? 'success' : 'muted'}>{badge}</Badge>
      </div>
      <p className="mt-2 text-sm text-muted">{desc}</p>
    </Card>
  );
}
