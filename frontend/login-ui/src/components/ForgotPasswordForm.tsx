'use client';

import Link from 'next/link';
import axios from 'axios';
import { useState } from 'react';
import toast from 'react-hot-toast';
import { api } from '@/lib/api';
import { Button, Input } from '@/components/ui';

export function ForgotPasswordForm({ appId }: { appId: string }) {
  const [email, setEmail] = useState('');
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState('');

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    try {
      const { data } = await api.post('/auth/forgot-password', { app_id: appId, email });
      setMessage(data.message);
    } catch (error) {
      if (axios.isAxiosError(error)) toast.error(error.response?.data?.error || 'Unable to send reset link');
    } finally {
      setLoading(false);
    }
  }

  return <div className="space-y-6"><form className="space-y-4" onSubmit={handleSubmit}><Input type="email" placeholder="Email" required value={email} onChange={(event) => setEmail(event.target.value)} /><Button type="submit" className="w-full justify-center" loading={loading}>Send Reset Link</Button></form>{message ? <p className="text-sm" style={{ color: 'var(--muted)' }}>{message}</p> : null}<Link href={`/?app_id=${encodeURIComponent(appId)}`} className="text-sm underline underline-offset-4" style={{ color: 'var(--foreground)' }}>Back to login</Link></div>;
}
