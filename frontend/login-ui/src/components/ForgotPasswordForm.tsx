'use client';

import Link from 'next/link';
import axios from 'axios';
import { useState } from 'react';
import toast from 'react-hot-toast';
import { api } from '@/lib/api';
import { Button, Input } from '@/components/ui';

export function ForgotPasswordForm({ appId, redirectUri }: { appId: string; redirectUri: string }) {
  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState('');

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    try {
      const { data } = await api.post('/auth/forgot-password', {
        app_id: appId,
        email,
        redirect_uri: redirectUri || undefined,
      });
      setMessage(data.message);
    } catch (error) {
      if (axios.isAxiosError(error)) {
        const code = error.response?.data?.error;
        toast.error(code === 'rate_limited' ? 'Too many attempts. Try again later.' : (code || 'Unable to send reset link'));
      }
    } finally {
      setLoading(false);
    }
  }

  const loginHref = redirectUri
    ? `/?app_id=${encodeURIComponent(appId)}&redirect_uri=${encodeURIComponent(redirectUri)}`
    : `/?app_id=${encodeURIComponent(appId)}`;

  return (
    <div className="space-y-6">
      <form className="space-y-4" onSubmit={handleSubmit}>
        <Input type="email" placeholder="Email" required value={email} onChange={(event) => setEmail(event.target.value)} />
        <Button type="submit" className="w-full justify-center" loading={loading}>Send Reset Link</Button>
      </form>
      {message ? <p className="text-sm" style={{ color: 'var(--muted)' }}>{message}</p> : null}
      <Link href={loginHref} className="text-sm underline underline-offset-4" style={{ color: 'var(--foreground)' }}>Back to login</Link>
    </div>
  );
}
