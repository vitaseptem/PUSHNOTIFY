'use client';

import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import type { TimeBucket } from '@/types';

export function DeliveryChart({ data }: { data: TimeBucket[] }) {
  const points = data.map((d) => ({
    time: new Date(d.bucket).toLocaleTimeString([], { hour: '2-digit' }),
    sent: d.sent,
    delivered: d.delivered,
  }));

  return (
    <ResponsiveContainer width="100%" height={280}>
      <AreaChart data={points} margin={{ top: 10, right: 10, left: -16, bottom: 0 }}>
        <defs>
          <linearGradient id="sentGrad" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#7B1C2E" stopOpacity={0.5} />
            <stop offset="95%" stopColor="#7B1C2E" stopOpacity={0} />
          </linearGradient>
          <linearGradient id="delGrad" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#22C55E" stopOpacity={0.4} />
            <stop offset="95%" stopColor="#22C55E" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="#1E1E2E" vertical={false} />
        <XAxis dataKey="time" stroke="#8B8BA0" fontSize={12} tickLine={false} axisLine={false} />
        <YAxis stroke="#8B8BA0" fontSize={12} tickLine={false} axisLine={false} allowDecimals={false} />
        <Tooltip
          contentStyle={{
            background: '#111118',
            border: '1px solid #1E1E2E',
            borderRadius: 12,
            color: '#F1F1F5',
          }}
        />
        <Area type="monotone" dataKey="sent" stroke="#9B2235" strokeWidth={2} fill="url(#sentGrad)" />
        <Area type="monotone" dataKey="delivered" stroke="#22C55E" strokeWidth={2} fill="url(#delGrad)" />
      </AreaChart>
    </ResponsiveContainer>
  );
}
