'use client';

import { useEffect, useState } from 'react';
import { Badge, Button, Card, EmptyState, ErrorState, Input, Modal, Skeleton } from '@/components/ui';
import { api } from '@/lib/api';
import { timeAgo } from '@/lib/utils';
import type { Paginated, Subscriber } from '@/types';

export default function SubscribersPage() {
  const [data, setData] = useState<Paginated<Subscriber> | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [open, setOpen] = useState(false);

  function load() {
    setLoading(true);
    api<Paginated<Subscriber>>('/api/v1/subscribers')
      .then(setData)
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }

  useEffect(load, []);
  const items = data?.data ?? [];

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button onClick={() => setOpen(true)}>Add subscriber</Button>
      </div>
      {error && <ErrorState message={error} />}
      <Card>
        {loading ? (
          <div className="space-y-2">
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-10" />
            ))}
          </div>
        ) : items.length === 0 ? (
          <EmptyState title="No subscribers yet" />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-left text-xs text-muted">
                <tr>
                  <th className="pb-2">External ID</th>
                  <th className="pb-2">Email</th>
                  <th className="pb-2">WhatsApp</th>
                  <th className="pb-2">Web Push</th>
                  <th className="pb-2">Created</th>
                </tr>
              </thead>
              <tbody>
                {items.map((s) => (
                  <tr key={s.id} className="border-t border-border">
                    <td className="py-2 font-mono text-xs">{s.external_id}</td>
                    <td className="py-2 text-muted">{s.email ?? '—'}</td>
                    <td className="py-2 text-muted">{s.whatsapp ?? '—'}</td>
                    <td className="py-2">
                      {s.web_push_subscription ? <Badge color="success">registered</Badge> : <span className="text-muted">—</span>}
                    </td>
                    <td className="py-2 text-muted">{timeAgo(s.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
      <AddSubscriberModal
        open={open}
        onClose={() => setOpen(false)}
        onAdded={() => {
          setOpen(false);
          load();
        }}
      />
    </div>
  );
}

function AddSubscriberModal({ open, onClose, onAdded }: { open: boolean; onClose: () => void; onAdded: () => void }) {
  const [externalId, setExternalId] = useState('');
  const [email, setEmail] = useState('');
  const [whatsapp, setWhatsapp] = useState('');
  const [phone, setPhone] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function submit() {
    setLoading(true);
    setError(null);
    try {
      await api('/api/v1/subscribers', {
        method: 'POST',
        body: {
          external_id: externalId,
          email: email || undefined,
          whatsapp: whatsapp || undefined,
          phone: phone || undefined,
        },
      });
      onAdded();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Add subscriber">
      <div className="space-y-4">
        {error && <ErrorState message={error} />}
        <Input placeholder="External ID (required)" value={externalId} onChange={(e) => setExternalId(e.target.value)} />
        <Input type="email" placeholder="Email" value={email} onChange={(e) => setEmail(e.target.value)} />
        <Input placeholder="WhatsApp (+5511…)" value={whatsapp} onChange={(e) => setWhatsapp(e.target.value)} />
        <Input placeholder="Phone (SMS)" value={phone} onChange={(e) => setPhone(e.target.value)} />
        <Button onClick={submit} loading={loading} disabled={!externalId} className="w-full">
          Save
        </Button>
      </div>
    </Modal>
  );
}
