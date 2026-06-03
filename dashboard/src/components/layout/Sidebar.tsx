'use client';

import Image from 'next/image';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { cn } from '@/lib/utils';

const nav = [
  { href: '/dashboard', label: 'Overview', icon: '◫' },
  { href: '/notifications', label: 'Notifications', icon: '✉' },
  { href: '/templates', label: 'Templates', icon: '❏' },
  { href: '/subscribers', label: 'Subscribers', icon: '⦿' },
  { href: '/channels', label: 'Channels', icon: '⇄' },
  { href: '/webhooks', label: 'Webhooks', icon: '↪' },
  { href: '/analytics', label: 'Analytics', icon: '◷' },
  { href: '/settings', label: 'Settings', icon: '⚙' },
];

export function Sidebar() {
  const pathname = usePathname();
  return (
    <aside className="hidden w-60 shrink-0 flex-col border-r border-border bg-surface md:flex">
      <div className="flex h-16 items-center gap-2 border-b border-border px-5">
        <Image src="/logo.png" alt="PushNotify" width={150} height={32} priority className="h-8 w-auto" />
      </div>
      <nav className="flex-1 space-y-1 p-3">
        {nav.map((item) => {
          const active = pathname === item.href || pathname.startsWith(item.href + '/');
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                'flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors',
                active ? 'bg-primary/20 text-foreground' : 'text-muted hover:bg-white/5 hover:text-foreground',
              )}
            >
              <span className="w-4 text-center">{item.icon}</span>
              {item.label}
            </Link>
          );
        })}
      </nav>
      <div className="border-t border-border p-4 text-xs text-muted">Built by Astraz Studio</div>
    </aside>
  );
}
