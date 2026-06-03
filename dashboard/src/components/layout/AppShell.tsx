'use client';

import { usePathname } from 'next/navigation';
import { Sidebar } from './Sidebar';
import { Header } from './Header';

const titles: Record<string, string> = {
  '/dashboard': 'Overview',
  '/notifications': 'Notifications',
  '/templates': 'Templates',
  '/subscribers': 'Subscribers',
  '/channels': 'Channels',
  '/webhooks': 'Webhooks',
  '/analytics': 'Analytics',
  '/settings': 'Settings',
};

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const key = Object.keys(titles).find((k) => pathname.startsWith(k));
  const title = key ? titles[key] : 'Dashboard';

  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header title={title ?? 'Dashboard'} />
        <main className="flex-1 overflow-y-auto p-6">{children}</main>
      </div>
    </div>
  );
}
