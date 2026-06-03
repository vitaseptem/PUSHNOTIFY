'use client';

import Image from 'next/image';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { FormEvent, useState } from 'react';
import { Button, Card, ErrorState, Input } from '@/components/ui';
import { register } from '@/hooks/useAuth';

export default function RegisterPage() {
  const router = useRouter();
  const [fullName, setFullName] = useState('');
  const [email, setEmail] = useState('');
  const [workspace, setWorkspace] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      await register(email, fullName, password, workspace || `${fullName}'s Workspace`);
      router.replace('/dashboard');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'registration failed');
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
        <h2 className="mb-1 text-center text-xl font-semibold">Create your workspace</h2>
        <p className="mb-6 text-center text-sm text-muted">Start sending notifications in minutes</p>
        <form onSubmit={onSubmit} className="space-y-4">
          {error && <ErrorState message={error} />}
          <Input placeholder="Full name" value={fullName} onChange={(e) => setFullName(e.target.value)} required />
          <Input type="email" placeholder="Email" value={email} onChange={(e) => setEmail(e.target.value)} required />
          <Input placeholder="Workspace name" value={workspace} onChange={(e) => setWorkspace(e.target.value)} />
          <Input
            type="password"
            placeholder="Password (min 8 chars)"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            minLength={8}
            required
          />
          <Button type="submit" loading={loading} className="w-full">
            Create account
          </Button>
        </form>
        <p className="mt-5 text-center text-sm text-muted">
          Already have an account?{' '}
          <Link href="/login" className="text-[#E98BA0] hover:underline">
            Sign in
          </Link>
        </p>
      </Card>
    </div>
  );
}
