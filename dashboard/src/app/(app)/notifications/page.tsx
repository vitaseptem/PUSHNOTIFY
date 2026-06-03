'use client';

import { useState } from 'react';
import { Badge, Button, Card, EmptyState, ErrorState, Input, Modal, Select, Skeleton } from '@/components/ui';
import { useNotifications } from '@/hooks/useNotifications';
import { api } from '@/lib/api';
import { shortId, statusColor, timeAgo } from '@/lib/utils';
import type { Channel, Delivery, Notification } from '@/types';

const ALL_CHANNELS: Channel[] = ['websocket', 'webpush', 'email', 'whatsapp', 'sms'];

export default function NotificationsPage() {
  const [status, setStatus] = useState('');
  const { data, loading, error, reload } = useNotifications(status);
  const [detail, setDetail] = useState<Notification | null>(null);
  const [deliveries, setDeliveries] = useState<Delivery[]>([]);
  const [sendOpen, setSendOpen] = useState(false);

  async function openDetail(n: Notification) {
    setDetail(n);
    const res = await api<{ data: Delivery[] }>(`/api/v1/notifications/${n.id}/deliveries`);
    setDeliveries(res.data ?? []);
  }

  const items = data?.data ?? [];

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <Select value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value="">All statuses</option>
          <option value="queued">Queued</option>
          <option value="sent">Sent</option>
          <option value="delivered">Delivered</option>
          <option value="failed">Failed</option>
          <option value="partial">Partial</option>
        </Select>
        <Button onClick={() => setSendOpen(true)}>Send test</Button>
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
          <EmptyState title="No notifications found" />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-left text-xs text-muted">
                <tr>
                  <th className="pb-2">ID</th>
                  <th className="pb-2">Channels</th>
                  <th className="pb-2">Status</th>
                  <th className="pb-2">Created</th>
                  <th className="pb-2"></th>
                </tr>
              </thead>
              <tbody>
                {items.map((n) => (
                  <tr key={n.id} className="border-t border-border">
                    <td className="py-2 font-mono text-xs">{shortId(n.id)}</td>
                    <td className="py-2">
                      <div className="flex flex-wrap gap-1">
                        {n.channels.map((c) => (
                          <Badge key={c} color="primary">
                            {c}
                          </Badge>
                        ))}
                      </div>
                    </td>
                    <td className="py-2">
                      <Badge color={statusColor(n.status)}>{n.status}</Badge>
                    </td>
                    <td className="py-2 text-muted">{timeAgo(n.created_at)}</td>
                    <td className="py-2 text-right">
                      <Button variant="ghost" onClick={() => openDetail(n)}>
                        Details
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      <Modal open={!!detail} onClose={() => setDetail(null)} title="Notification details">
        {detail && (
          <div className="space-y-4 text-sm">
            <div>
              <p className="text-xs text-muted">ID</p>
              <p className="font-mono text-xs">{detail.id}</p>
            </div>
            <div>
              <p className="mb-2 text-xs text-muted">Deliveries</p>
              <div className="space-y-2">
                {deliveries.map((d) => (
                  <div key={d.id} className="flex items-center justify-between rounded-lg border border-border p-3">
                    <div>
                      <Badge color="primary">{d.channel}</Badge>
                      {d.error_message && <p className="mt-1 text-xs text-error">{d.error_message}</p>}
                    </div>
                    <div className="text-right">
                      <Badge color={statusColor(d.status)}>{d.status}</Badge>
                      <p className="mt-1 text-xs text-muted">
                        {d.attempts}/{d.max_attempts} attempts
                      </p>
                    </div>
                  </div>
                ))}
                {deliveries.length === 0 && <p className="text-muted">No deliveries.</p>}
              </div>
            </div>
          </div>
        )}
      </Modal>

      <SendTestModal
        open={sendOpen}
        onClose={() => setSendOpen(false)}
        onSent={() => {
          setSendOpen(false);
          reload();
        }}
      />
    </div>
  );
}

function SendTestModal({ open, onClose, onSent }: { open: boolean; onClose: () => void; onSent: () => void }) {
  const [subscriberId, setSubscriberId] = useState('');
  const [subject, setSubject] = useState('');
  const [body, setBody] = useState('');
  const [channels, setChannels] = useState<Channel[]>(['websocket']);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  function toggle(c: Channel) {
    setChannels((prev) => (prev.includes(c) ? prev.filter((x) => x !== c) : [...prev, c]));
  }

  async function submit() {
    setLoading(true);
    setError(null);
    try {
      await api('/api/v1/notifications/send', {
        method: 'POST',
        body: { subscriber_id: subscriberId, subject, body, channels },
      });
      onSent();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'send failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Send a test notification">
      <div className="space-y-4">
        {error && <ErrorState message={error} />}
        <Input placeholder="Subscriber external ID" value={subscriberId} onChange={(e) => setSubscriberId(e.target.value)} />
        <Input placeholder="Subject" value={subject} onChange={(e) => setSubject(e.target.value)} />
        <Input placeholder="Body" value={body} onChange={(e) => setBody(e.target.value)} />
        <div className="flex flex-wrap gap-2">
          {ALL_CHANNELS.map((c) => (
            <button
              key={c}
              type="button"
              onClick={() => toggle(c)}
              className={`rounded-md px-3 py-1 text-xs ${
                channels.includes(c) ? 'bg-primary text-white' : 'bg-white/5 text-muted'
              }`}
            >
              {c}
            </button>
          ))}
        </div>
        <Button onClick={submit} loading={loading} disabled={!subscriberId || !body} className="w-full">
          Send
        </Button>
      </div>
    </Modal>
  );
}
