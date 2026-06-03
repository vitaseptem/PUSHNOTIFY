'use client';

import { Card } from '@/components/ui';
import { cn } from '@/lib/utils';

export function KpiCard({
  label,
  value,
  delta,
  tone = 'default',
}: {
  label: string;
  value: string;
  delta?: string;
  tone?: 'default' | 'success' | 'error';
}) {
  return (
    <Card>
      <p className="text-sm text-muted">{label}</p>
      <p
        className={cn(
          'mt-2 font-mono text-3xl font-semibold',
          tone === 'success' && 'text-success',
          tone === 'error' && 'text-error',
        )}
      >
        {value}
      </p>
      {delta && <p className="mt-1 text-xs text-muted">{delta}</p>}
    </Card>
  );
}
