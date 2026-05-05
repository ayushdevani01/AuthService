'use client';

import Link from 'next/link';
import axios from 'axios';
import { useEffect, useRef, useState } from 'react';
import { api } from '@/lib/api';

export function VerifyEmailView({ appId, token, redirectUri }: { appId: string; token: string; redirectUri: string }) {
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading');
  const [message, setMessage] = useState('');
  const requested = useRef(false);

  useEffect(() => {
    if (requested.current) return;
    requested.current = true;

    async function verify() {
      try {
        const { data } = await api.post('/auth/verify-email', { app_id: appId, token });
        setMessage(data.message);
        setStatus('success');
      } catch (error) {
        if (axios.isAxiosError(error)) setMessage(error.response?.data?.error || 'Verification failed');
        setStatus('error');
      }
    }
    void verify();
  }, [appId, token]);

  const loginHref = redirectUri
    ? `/?app_id=${encodeURIComponent(appId)}&redirect_uri=${encodeURIComponent(redirectUri)}`
    : `/?app_id=${encodeURIComponent(appId)}`;

  if (status === 'loading') return <p className="text-sm" style={{ color: 'var(--muted)' }}>Verifying your email...</p>;
  return (
    <div className="space-y-4">
      <p className="text-sm" style={{ color: 'var(--foreground)' }}>{message}</p>
      <Link href={loginHref} className="text-sm underline underline-offset-4" style={{ color: 'var(--foreground)' }}>Go to login</Link>
    </div>
  );
}
