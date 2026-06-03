'use client';

import { AppShell } from '@/components/layout/AppShell';
import { useAuth } from '@/hooks/useAuth';

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const { loading, user } = useAuth();

  if (loading) {
    return (
      <div className="flex h-screen items-center justify-center text-muted">
        <span className="h-5 w-5 animate-spin rounded-full border-2 border-border border-t-primary" />
      </div>
    );
  }
  if (!user) return null; // redirect handled by useAuth

  return <AppShell>{children}</AppShell>;
}
