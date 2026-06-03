'use client';

import { useCallback, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { api, clearToken, getToken, setToken } from '@/lib/api';
import type { AuthResponse, User, Workspace } from '@/types';

interface MeResponse {
  user: User;
  workspace: Workspace;
}

/** Loads the current session and exposes auth actions. */
export function useAuth(redirectIfUnauthenticated = true) {
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const token = getToken();
    if (!token) {
      setLoading(false);
      if (redirectIfUnauthenticated) router.replace('/login');
      return;
    }
    api<MeResponse>('/api/v1/auth/me')
      .then((me) => {
        setUser(me.user);
        setWorkspace(me.workspace);
      })
      .catch(() => {
        clearToken();
        if (redirectIfUnauthenticated) router.replace('/login');
      })
      .finally(() => setLoading(false));
  }, [router, redirectIfUnauthenticated]);

  const logout = useCallback(() => {
    clearToken();
    router.replace('/login');
  }, [router]);

  return { user, workspace, loading, logout };
}

/** Authenticates against /auth/login and stores the token. */
export async function login(email: string, password: string): Promise<AuthResponse> {
  const res = await api<AuthResponse>('/api/v1/auth/login', {
    method: 'POST',
    auth: false,
    body: { email, password },
  });
  setToken(res.token);
  return res;
}

/** Registers a new account and stores the token. */
export async function register(
  email: string,
  fullName: string,
  password: string,
  workspaceName: string,
): Promise<AuthResponse> {
  const res = await api<AuthResponse>('/api/v1/auth/register', {
    method: 'POST',
    auth: false,
    body: { email, full_name: fullName, password, workspace_name: workspaceName },
  });
  setToken(res.token);
  return res;
}
