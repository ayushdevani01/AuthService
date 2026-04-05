'use client';

import Link from 'next/link';
import axios from 'axios';
import { useState } from 'react';
import toast from 'react-hot-toast';
import { api } from '@/lib/api';
import { createPkcePair } from '@/lib/pkce';
import { Button, Input } from '@/components/ui';

export function LoginForm({ appId, redirectUri }: { appId: string; redirectUri: string }) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [oauthLoading, setOauthLoading] = useState<string | null>(null);

  async function handleEmailLogin(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    try {
      const { data } = await api.post('/auth/login', { app_id: appId, email, password });
      const target = new URL(redirectUri);
      const hash = new URLSearchParams();
      hash.set('access_token', data.access_token);
      hash.set('refresh_token', data.refresh_token);
      hash.set('token_type', data.token_type);
      hash.set('expires_at', String(data.expires_at));
      target.hash = hash.toString();
      window.location.href = target.toString();
    } catch (error) {
      if (axios.isAxiosError(error)) {
        const response = error.response?.data;
        if (response?.requires_verification) {
          toast.error('Please verify your email before signing in. Check your inbox.');
          return;
        }
        toast.error(response?.error || 'Unable to sign in');
      }
    } finally {
      setLoading(false);
    }
  }

  async function handleOAuth(provider: 'google' | 'github') {
    setOauthLoading(provider);
    try {
      const { verifier, challenge } = await createPkcePair();
      sessionStorage.setItem('code_verifier', verifier);
      const params = new URLSearchParams({
        app_id: appId,
        provider,
        redirect_uri: redirectUri,
        code_challenge: challenge,
        code_challenge_method: 'S256',
        code_verifier: verifier,
      });
      window.location.href = `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/oauth/authorize?${params.toString()}`;
    } finally {
      setOauthLoading(null);
    }
  }

  return (
    <div className="space-y-6">
      <div className="grid gap-3">
        <Button variant="secondary" onClick={() => handleOAuth('google')} loading={oauthLoading === 'google'} className="w-full justify-center">Continue with Google</Button>
        <Button variant="secondary" onClick={() => handleOAuth('github')} loading={oauthLoading === 'github'} className="w-full justify-center">Continue with GitHub</Button>
      </div>
      <div className="flex items-center gap-3 text-xs uppercase tracking-[0.35em]" style={{ color: 'var(--muted)' }}><span className="h-px flex-1" style={{ background: 'var(--border)' }} />or<span className="h-px flex-1" style={{ background: 'var(--border)' }} /></div>
      <form className="space-y-4" onSubmit={handleEmailLogin}>
        <Input type="email" placeholder="Email" required value={email} onChange={(event) => setEmail(event.target.value)} />
        <Input type="password" placeholder="Password" required value={password} onChange={(event) => setPassword(event.target.value)} />
        <Button type="submit" className="w-full justify-center" loading={loading}>Sign In</Button>
      </form>
      <div className="flex items-center justify-between text-sm" style={{ color: 'var(--muted)' }}>
        <Link href={`/?app_id=${encodeURIComponent(appId)}&redirect_uri=${encodeURIComponent(redirectUri)}&mode=register`} className="underline underline-offset-4" style={{ color: 'var(--foreground)' }}>Create account</Link>
        <Link href={`/?app_id=${encodeURIComponent(appId)}&mode=forgot`} className="underline underline-offset-4" style={{ color: 'var(--foreground)' }}>Forgot password</Link>
      </div>
    </div>
  );
}
