'use client';

import { useEffect, useRef, useState } from 'react';
import { apiBaseUrl, getToken } from '@/lib/api';

interface LivePayload {
  connected_now: number;
  timestamp: string;
}

/**
 * Subscribes to the server's SSE live-metrics stream. EventSource cannot send
 * an Authorization header, so we stream via fetch + ReadableStream instead.
 */
export function useLive() {
  const [connected, setConnected] = useState(false);
  const [data, setData] = useState<LivePayload | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    const token = getToken();
    if (!token) return;

    const controller = new AbortController();
    abortRef.current = controller;
    let cancelled = false;

    (async () => {
      try {
        const res = await fetch(`${apiBaseUrl}/api/v1/dashboard/live`, {
          headers: { Authorization: `Bearer ${token}` },
          signal: controller.signal,
        });
        if (!res.body) return;
        setConnected(true);

        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        while (!cancelled) {
          const { done, value } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          const events = buffer.split('\n\n');
          buffer = events.pop() ?? '';
          for (const evt of events) {
            const line = evt.split('\n').find((l) => l.startsWith('data:'));
            if (!line) continue;
            try {
              setData(JSON.parse(line.slice(5).trim()) as LivePayload);
            } catch {
              // ignore malformed frame
            }
          }
        }
      } catch {
        if (!cancelled) setConnected(false);
      } finally {
        if (!cancelled) setConnected(false);
      }
    })();

    return () => {
      cancelled = true;
      controller.abort();
    };
  }, []);

  return { connected, data };
}
