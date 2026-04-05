'use client';

import Link from 'next/link';
import axios from 'axios';
import { useState } from 'react';
import toast from 'react-hot-toast';
import { api } from '@/lib/api';
import { Button, Input } from '@/components/ui';

export function RegisterForm({ appId, redirectUri }: { appId: string; redirectUri: string }) {
  const [form, setForm] = useState({ name: '', email: '', confirmEmail: '', password: '' });
  const [loading, setLoading] = useState(false);
  const [verificationMessage, setVerificationMessage] = useState('');

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (form.email.trim().toLowerCase() !== form.confirmEmail.trim().toLowerCase()) {
      toast.error('Email and confirm email must match');
      return;
    }

    setLoading(true);
    try {
      const { data } = await api.post('/auth/register', { app_id: appId, name: form.name, email: form.email, password: form.password });

      if (data.requires_verification) {
        const message = data.message || 'Account created. Check your inbox and verify your email before signing in.';
        setVerificationMessage(message);
        toast.success(message);
        return;
      }

      if (!data.access_token) {
        toast.error('Registration completed, but no session was created. Please sign in after verification.');
        return;
      }

      const target = new URL(redirectUri);
      const hash = new URLSearchParams();
      hash.set('access_token', data.access_token);
      hash.set('refresh_token', data.refresh_token || '');
      hash.set('token_type', data.token_type);
      hash.set('expires_at', String(data.expires_at));
      target.hash = hash.toString();
      window.location.href = target.toString();
    } catch (error) {
      if (axios.isAxiosError(error)) toast.error(error.response?.data?.error || 'Unable to create account');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-6">
      <form className="space-y-4" onSubmit={handleSubmit}>
        <Input placeholder="Name" value={form.name} onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))} />
        <Input type="email" placeholder="Email" required value={form.email} onChange={(event) => setForm((current) => ({ ...current, email: event.target.value }))} />
        <Input type="email" placeholder="Confirm Email" required value={form.confirmEmail} onChange={(event) => setForm((current) => ({ ...current, confirmEmail: event.target.value }))} />
        <Input type="password" placeholder="Password" required value={form.password} onChange={(event) => setForm((current) => ({ ...current, password: event.target.value }))} />
        <Button type="submit" className="w-full justify-center" loading={loading}>Create Account</Button>
      </form>
      {verificationMessage ? <div className="rounded-3xl border border-[var(--border)] bg-[var(--background-alt)] p-4 text-sm leading-6" style={{ color: 'var(--muted)' }}>{verificationMessage}</div> : null}
      <p className="text-sm" style={{ color: 'var(--muted)' }}>Already have access? <Link href={`/?app_id=${encodeURIComponent(appId)}&redirect_uri=${encodeURIComponent(redirectUri)}`} className="underline underline-offset-4" style={{ color: 'var(--foreground)' }}>Sign in</Link></p>
    </div>
  );
}
