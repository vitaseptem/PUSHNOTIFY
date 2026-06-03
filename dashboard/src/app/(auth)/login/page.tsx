'use client';

import Image from 'next/image';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { FormEvent, useState } from 'react';
import { Button, Card, ErrorState, Input } from '@/components/ui';
import { login } from '@/hooks/useAuth';

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      await login(email, password);
      router.replace('/dashboard');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'login failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <Card className="w-full max-w-sm">
        <div className="mb-6 flex justify-center">
          <Image src="/logo.png" alt="PushNotify" width={170} height={36} priority className="h-9 w-auto" />
        </div>
        <h2 className="mb-1 text-center text-xl font-semibold">Welcome back</h2>
        <p className="mb-6 text-center text-sm text-muted">Sign in to your workspace</p>
        <form onSubmit={onSubmit} className="space-y-4">
          {error && <ErrorState message={error} />}
          <Input type="email" placeholder="Email" value={email} onChange={(e) => setEmail(e.target.value)} required />
          <Input
            type="password"
            placeholder="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
          <Button type="submit" loading={loading} className="w-full">
            Sign in
          </Button>
        </form>
        <p className="mt-5 text-center text-sm text-muted">
          No account?{' '}
          <Link href="/register" className="text-[#E98BA0] hover:underline">
            Create one
          </Link>
        </p>
      </Card>
    </div>
  );
}
