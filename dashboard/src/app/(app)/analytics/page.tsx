'use client';

import { useEffect, useState } from 'react';
import { Card, ErrorState, Select, Skeleton } from '@/components/ui';
import { DeliveryChart } from '@/components/charts/DeliveryChart';
import { api } from '@/lib/api';
import { formatNumber } from '@/lib/utils';
import type { ChannelTotals, TimeBucket } from '@/types';

interface AnalyticsResponse {
  period: string;
  totals: { sent: number; delivered: number; failed: number };
  series: TimeBucket[] | null;
  by_channel: ChannelTotals[] | null;
}

export default function AnalyticsPage() {
  const [period, setPeriod] = useState('7d');
  const [data, setData] = useState<AnalyticsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    api<AnalyticsResponse>(`/api/v1/dashboard/analytics?period=${period}`)
      .then(setData)
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, [period]);

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Select value={period} onChange={(e) => setPeriod(e.target.value)}>
          <option value="24h">Last 24 hours</option>
          <option value="7d">Last 7 days</option>
          <option value="30d">Last 30 days</option>
        </Select>
      </div>
      {error && <ErrorState message={error} />}
      {loading || !data ? (
        <Skeleton className="h-80" />
      ) : (
        <>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <Stat label="Sent" value={data.totals.sent} />
            <Stat label="Delivered" value={data.totals.delivered} tone="success" />
            <Stat label="Failed" value={data.totals.failed} tone="error" />
          </div>
          <Card>
            <h2 className="mb-4 text-sm font-medium text-muted">Delivery trend</h2>
            <DeliveryChart data={data.series ?? []} />
          </Card>
          <Card>
            <h2 className="mb-4 text-sm font-medium text-muted">By channel</h2>
            <div className="space-y-2 text-sm">
              {(data.by_channel ?? []).map((c) => (
                <div key={c.channel} className="flex items-center justify-between border-t border-border py-2 first:border-0">
                  <span className="capitalize">{c.channel}</span>
                  <span className="text-muted">
                    {formatNumber(c.delivered)} / {formatNumber(c.sent)} delivered
                  </span>
                </div>
              ))}
              {(data.by_channel ?? []).length === 0 && <p className="text-muted">No data for this period.</p>}
            </div>
          </Card>
        </>
      )}
    </div>
  );
}

function Stat({ label, value, tone }: { label: string; value: number; tone?: 'success' | 'error' }) {
  return (
    <Card>
      <p className="text-sm text-muted">{label}</p>
      <p
        className={`mt-2 font-mono text-2xl font-semibold ${
          tone === 'success' ? 'text-success' : tone === 'error' ? 'text-error' : ''
        }`}
      >
        {formatNumber(value)}
      </p>
    </Card>
  );
}
