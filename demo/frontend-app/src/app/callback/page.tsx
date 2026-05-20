'use client';

import { AuthCallbackHandler, AuthServiceProvider, useAuth } from 'authservice-react';
import { useEffect } from 'react';
import { useRouter } from 'next/navigation';

function CallbackInner() {
  const router = useRouter();
  const { isAuthenticated, loading } = useAuth();

  useEffect(() => {
    if (!loading && isAuthenticated) {
      router.replace('/');
    }
  }, [isAuthenticated, loading, router]);

  return <main><p>Completing sign-in…</p></main>;
}

export default function CallbackPage() {
  const appId = process.env.NEXT_PUBLIC_AUTH_PUBLISHABLE_KEY || '';
  const authUrl = process.env.NEXT_PUBLIC_AUTH_URL || 'http://localhost:3001';
  const redirectUri = process.env.NEXT_PUBLIC_AUTH_REDIRECT_URI || 'http://localhost:3000/callback';
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

  if (!appId) {
    return <main><p>Missing NEXT_PUBLIC_AUTH_PUBLISHABLE_KEY</p></main>;
  }

  return (
    <AuthServiceProvider appId={appId} authUrl={authUrl} redirectUri={redirectUri} apiUrl={apiUrl}>
      <AuthCallbackHandler />
      <CallbackInner />
    </AuthServiceProvider>
  );
}
