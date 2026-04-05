'use client';

import { useEffect, useMemo } from 'react';
import { useRouter } from 'next/navigation';

function parseFragment(hash: string) {
  const fragment = hash.startsWith('#') ? hash.slice(1) : hash;
  return new URLSearchParams(fragment);
}

export default function AuthCallbackPage() {
  const router = useRouter();

  const tokens = useMemo(() => {
    if (typeof window === 'undefined') return null;
    const params = parseFragment(window.location.hash);
    return {
      accessToken: params.get('access_token'),
      refreshToken: params.get('refresh_token'),
      tokenType: params.get('token_type'),
      expiresAt: params.get('expires_at'),
    };
  }, []);

  useEffect(() => {
    if (!tokens?.accessToken) {
      return;
    }

    sessionStorage.setItem(
      'authservice_callback_tokens',
      JSON.stringify({
        access_token: tokens.accessToken,
        refresh_token: tokens.refreshToken,
        token_type: tokens.tokenType,
        expires_at: tokens.expiresAt,
      }),
    );

    router.replace('/');
  }, [router, tokens]);

  return <main className="flex min-h-screen items-center justify-center text-sm text-zinc-400">Finalizing sign-in…</main>;
}
