'use client';

import { useEffect, useState } from 'react';
import { Badge, Card, EmptyState, ErrorState, Skeleton } from '@/components/ui';
import { KpiCard } from '@/components/dashboard/KpiCard';
import { DeliveryChart } from '@/components/charts/DeliveryChart';
import { api } from '@/lib/api';
import { formatNumber, formatPercent, shortId, statusColor, timeAgo } from '@/lib/utils';
import type { Notification, Overview, Paginated } from '@/types';

const CHANNELS: Array<{ key: string; label: string }> = [
  { key: 'websocket', label: 'WebSocket' },
  { key: 'webpush', label: 'Web Push' },
  { key: 'email', label: 'Email' },
  { key: 'whatsapp', label: 'WhatsApp' },
  { key: 'sms', label: 'SMS' },
];

export default function DashboardPage() {
  const [overview, setOverview] = useState<Overview | null>(null);
  const [recent, setRecent] = useState<Notification[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([
      api<Overview>('/api/v1/dashboard/overview'),
      api<Paginated<Notification>>('/api/v1/notifications?per_page=10'),
    ])
      .then(([ov, notifs]) => {
        setOverview(ov);
        setRecent(notifs.data ?? []);
      })
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-28" />
        ))}
        <Skeleton className="col-span-full h-72" />
      </div>
    );
  }

  if (error) return <ErrorState message={error} />;
  if (!overview) return null;

  const byChannel = overview.by_channel ?? [];
  const channelMap = new Map(byChannel.map((c) => [c.channel, c]));

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard label="Notifications sent (24h)" value={formatNumber(overview.total_sent_24h)} />
        <KpiCard
          label="Delivery rate"
          value={formatPercent(overview.delivery_rate)}
          tone={overview.delivery_rate >= 90 ? 'success' : 'error'}
        />
        <KpiCard
          label="Failures (24h)"
          value={formatNumber(overview.total_failed)}
          tone={overview.total_failed > 0 ? 'error' : 'default'}
        />
        <KpiCard label="Connected now" value={formatNumber(overview.connected_now)} delta="live via WebSocket" />
      </div>

      <Card>
        <h2 className="mb-4 text-sm font-medium text-muted">Sent vs Delivered · last 24h</h2>
        {overview.hourly_chart && overview.hourly_chart.length > 0 ? (
          <DeliveryChart data={overview.hourly_chart} />
        ) : (
          <EmptyState title="No delivery data yet" hint="Send your first notification to see metrics." />
        )}
      </Card>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-5">
        {CHANNELS.map((ch) => {
          const stats = channelMap.get(ch.key as never);
          return (
            <Card key={ch.key}>
              <p className="text-sm font-medium">{ch.label}</p>
              <div className="mt-3 space-y-1 text-xs">
                <Row label="Sent" value={stats?.sent ?? 0} />
                <Row label="Delivered" value={stats?.delivered ?? 0} tone="success" />
                <Row label="Failed" value={stats?.failed ?? 0} tone="error" />
              </div>
            </Card>
          );
        })}
      </div>

      <Card>
        <h2 className="mb-4 text-sm font-medium text-muted">Latest notifications</h2>
        {recent.length === 0 ? (
          <EmptyState title="No notifications yet" />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-left text-xs text-muted">
                <tr>
                  <th className="pb-2">ID</th>
                  <th className="pb-2">Channels</th>
                  <th className="pb-2">Status</th>
                  <th className="pb-2">Created</th>
                </tr>
              </thead>
              <tbody>
                {recent.map((n) => (
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
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}

function Row({ label, value, tone }: { label: string; value: number; tone?: 'success' | 'error' }) {
  return (
    <div className="flex justify-between">
      <span className="text-muted">{label}</span>
      <span className={tone === 'success' ? 'text-success' : tone === 'error' ? 'text-error' : 'text-foreground'}>
        {formatNumber(value)}
      </span>
    </div>
  );
}
