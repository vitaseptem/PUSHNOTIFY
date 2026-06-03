'use client';

import { useEffect, useState } from 'react';
import { Badge, Button, Card, ErrorState, Input, Skeleton } from '@/components/ui';
import { cn, formatNumber } from '@/lib/utils';
import { api } from '@/lib/api';
import type { APIKey, Workspace } from '@/types';

type Tab = 'workspace' | 'api-keys' | 'plan';

export default function SettingsPage() {
  const [tab, setTab] = useState<Tab>('workspace');
  return (
    <div className="space-y-4">
      <div className="flex gap-2">
        {(['workspace', 'api-keys', 'plan'] as Tab[]).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={cn(
              'rounded-lg px-3 py-1.5 text-sm capitalize',
              tab === t ? 'bg-primary text-white' : 'bg-surface text-muted hover:text-foreground',
            )}
          >
            {t.replace('-', ' ')}
          </button>
        ))}
      </div>
      {tab === 'workspace' && <WorkspaceTab />}
      {tab === 'api-keys' && <ApiKeysTab />}
      {tab === 'plan' && <PlanTab />}
    </div>
  );
}

function WorkspaceTab() {
  const [ws, setWs] = useState<Workspace | null>(null);
  const [name, setName] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api<Workspace>('/api/v1/workspace')
      .then((w) => {
        setWs(w);
        setName(w.name);
      })
      .catch((e: Error) => setError(e.message));
  }, []);

  async function save() {
    setSaving(true);
    setError(null);
    try {
      const updated = await api<Workspace>('/api/v1/workspace', { method: 'PUT', body: { name } });
      setWs(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'save failed');
    } finally {
      setSaving(false);
    }
  }

  if (!ws) return <Skeleton className="h-32" />;
  return (
    <Card className="max-w-lg space-y-4">
      {error && <ErrorState message={error} />}
      <div>
        <label className="text-sm text-muted">Workspace name</label>
        <Input value={name} onChange={(e) => setName(e.target.value)} className="mt-1" />
      </div>
      <div>
        <label className="text-sm text-muted">Slug</label>
        <p className="mt-1 font-mono text-sm">{ws.slug}</p>
      </div>
      <Button onClick={save} loading={saving}>
        Save changes
      </Button>
    </Card>
  );
}

function ApiKeysTab() {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [name, setName] = useState('');
  const [created, setCreated] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  function load() {
    api<{ data: APIKey[] | null }>('/api/v1/workspace/api-keys')
      .then((res) => setKeys(res.data ?? []))
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }
  useEffect(load, []);

  async function create() {
    setError(null);
    try {
      const key = await api<APIKey>('/api/v1/workspace/api-keys', { method: 'POST', body: { name } });
      setCreated(key.key ?? null);
      setName('');
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'create failed');
    }
  }

  async function revoke(id: string) {
    await api(`/api/v1/workspace/api-keys/${id}`, { method: 'DELETE' });
    load();
  }

  return (
    <div className="max-w-2xl space-y-4">
      {error && <ErrorState message={error} />}
      {created && (
        <Card className="border-success/30 bg-success/10">
          <p className="text-sm text-success">New API key (shown only once):</p>
          <p className="mt-2 break-all font-mono text-xs">{created}</p>
        </Card>
      )}
      <Card className="flex items-end gap-3">
        <div className="flex-1">
          <label className="text-sm text-muted">New key name</label>
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="production" className="mt-1" />
        </div>
        <Button onClick={create} disabled={!name}>
          Generate
        </Button>
      </Card>
      {loading ? (
        <Skeleton className="h-24" />
      ) : (
        <div className="space-y-2">
          {keys.map((k) => (
            <Card key={k.id} className="flex items-center justify-between">
              <div>
                <p className="font-medium">{k.name}</p>
                <p className="font-mono text-xs text-muted">{k.key_prefix}…</p>
              </div>
              <div className="flex items-center gap-3">
                <Badge color={k.is_active ? 'success' : 'muted'}>{k.is_active ? 'active' : 'revoked'}</Badge>
                <Button variant="danger" onClick={() => revoke(k.id)}>
                  Revoke
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

function PlanTab() {
  const [ws, setWs] = useState<Workspace | null>(null);
  useEffect(() => {
    api<Workspace>('/api/v1/workspace').then(setWs).catch(() => undefined);
  }, []);
  if (!ws) return <Skeleton className="h-32" />;
  const pct = ws.notifications_limit > 0 ? (ws.notifications_sent / ws.notifications_limit) * 100 : 0;
  return (
    <Card className="max-w-lg space-y-4">
      <div className="flex items-center justify-between">
        <p className="font-medium capitalize">{ws.plan} plan</p>
        <Badge color="primary">{ws.plan}</Badge>
      </div>
      <div>
        <div className="mb-1 flex justify-between text-sm">
          <span className="text-muted">Usage</span>
          <span>
            {formatNumber(ws.notifications_sent)} / {formatNumber(ws.notifications_limit)}
          </span>
        </div>
        <div className="h-2 overflow-hidden rounded-full bg-background">
          <div className="h-full bg-primary" style={{ width: `${Math.min(100, pct)}%` }} />
        </div>
      </div>
    </Card>
  );
}
