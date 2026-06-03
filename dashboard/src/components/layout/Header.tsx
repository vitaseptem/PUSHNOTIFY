'use client';

import { cn } from '@/lib/utils';
import { Button } from '@/components/ui';
import { useAuth } from '@/hooks/useAuth';
import { useLive } from '@/hooks/useLive';

export function Header({ title }: { title: string }) {
  const { user, logout } = useAuth();
  const { connected } = useLive();

  return (
    <header className="flex h-16 items-center justify-between border-b border-border px-6">
      <div className="flex items-center gap-3">
        <h1 className="text-lg font-semibold">{title}</h1>
        <span className="flex items-center gap-1.5 text-xs text-muted">
          <span
            className={cn(
              'h-2 w-2 rounded-full',
              connected ? 'bg-success animate-pulse-live' : 'bg-muted',
            )}
          />
          {connected ? 'LIVE' : 'offline'}
        </span>
      </div>
      <div className="flex items-center gap-3">
        {user && <span className="hidden text-sm text-muted sm:block">{user.email}</span>}
        <Button variant="ghost" onClick={logout}>
          Sign out
        </Button>
      </div>
    </header>
  );
}
