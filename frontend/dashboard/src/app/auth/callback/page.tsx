'use client';

import Link from 'next/link';
import { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import { useAuthStore } from '@/store/auth';
import type { Developer } from '@/lib/types';

function parseFragment(hash: string) {
  const fragment = hash.startsWith('#') ? hash.slice(1) : hash;
  return new URLSearchParams(fragment);
}

export default function AuthCallbackPage() {
  const router = useRouter();
  const setAuth = useAuthStore((state) => state.setAuth);
  const [status, setStatus] = useState<'working' | 'developer' | 'tokens' | 'error'>('working');
  const [error, setError] = useState('');

  const tokens = useMemo(() => {
    if (typeof window === 'undefined') return null;
    const params = parseFragment(window.location.hash);
    return {
      accessToken: params.get('access_token'),
      refreshToken: params.get('refresh_token'),
      tokenType: params.get('token_type'),
      expiresAt: params.get('expires_at'),
      error: params.get('error'),
    };
  }, []);

  useEffect(() => {
    if (!tokens) return;

    if (tokens.error) {
      setError(tokens.error);
      setStatus('error');
      if (typeof window !== 'undefined') {
        window.history.replaceState(null, '', window.location.pathname);
      }
      return;
    }

    if (!tokens.accessToken) {
      setError('missing_tokens');
      setStatus('error');
      return;
    }

    let cancelled = false;

    async function hydrate() {
      sessionStorage.setItem(
        'authservice_callback_tokens',
        JSON.stringify({
          access_token: tokens!.accessToken,
          refresh_token: tokens!.refreshToken,
          token_type: tokens!.tokenType,
          expires_at: tokens!.expiresAt,
        }),
      );

      if (typeof window !== 'undefined') {
        window.history.replaceState(null, '', window.location.pathname);
      }

      try {
        const { data } = await api.get('/api/v1/developers/profile', {
          headers: { Authorization: `Bearer ${tokens!.accessToken}` },
        });
        if (cancelled) return;
        setAuth({
          accessToken: tokens!.accessToken as string,
          developer: (data.developer || data) as Developer,
        });
        setStatus('developer');
        router.replace('/dashboard');
      } catch {
        if (cancelled) return;
        // End-user app tokens land here for demos; keep them in sessionStorage.
        setStatus('tokens');
      }
    }

    void hydrate();
    return () => {
      cancelled = true;
    };
  }, [router, setAuth, tokens]);

  if (status === 'working' || status === 'developer') {
    return <main className="flex min-h-screen items-center justify-center text-sm text-zinc-400">Finalizing sign-in…</main>;
  }

  if (status === 'error') {
    return (
      <main className="flex min-h-screen flex-col items-center justify-center gap-4 px-4 text-sm text-zinc-400">
        <p>Sign-in failed{error ? `: ${error}` : ''}.</p>
        <Link href="/login" className="underline underline-offset-4 text-zinc-200">Back to login</Link>
      </main>
    );
  }

  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-4 px-4 text-sm text-zinc-400">
      <p className="text-zinc-200">Tokens received and stored for this browser session.</p>
      <p>Hash fragment cleared. Read `authservice_callback_tokens` from sessionStorage in your app, or continue in the dashboard.</p>
      <div className="flex gap-4">
        <Link href="/dashboard" className="underline underline-offset-4 text-zinc-200">Open dashboard</Link>
        <Link href="/login" className="underline underline-offset-4 text-zinc-200">Developer login</Link>
      </div>
    </main>
  );
}
