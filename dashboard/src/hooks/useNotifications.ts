'use client';

import { useCallback, useEffect, useState } from 'react';
import { api } from '@/lib/api';
import type { Notification, Paginated } from '@/types';

/** Fetches a page of notifications with optional status filtering. */
export function useNotifications(status = '', page = 1) {
  const [data, setData] = useState<Paginated<Notification> | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    const params = new URLSearchParams();
    if (status) params.set('status', status);
    params.set('page', String(page));
    api<Paginated<Notification>>(`/api/v1/notifications?${params.toString()}`)
      .then((res) => {
        setData(res);
        setError(null);
      })
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, [status, page]);

  useEffect(() => {
    load();
  }, [load]);

  return { data, loading, error, reload: load };
}
