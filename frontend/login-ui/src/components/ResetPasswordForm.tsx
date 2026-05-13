'use client';

import Link from 'next/link';
import axios from 'axios';
import { useState } from 'react';
import toast from 'react-hot-toast';
import { api } from '@/lib/api';
import { Button, Input } from '@/components/ui';

export function ResetPasswordForm({ appId, token, redirectUri }: { appId: string; token: string; redirectUri: string }) {
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [done, setDone] = useState(false);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (password !== confirmPassword) {
      toast.error('Passwords do not match');
      return;
    }
    if (password.length < 8) {
      toast.error('Password must be at least 8 characters');
      return;
    }
    if (password.length > 72) {
      toast.error('Password must be at most 72 characters');
      return;
    }
    setLoading(true);
    try {
      await api.post('/auth/reset-password', {
        app_id: appId,
        token,
        new_password: password,
        confirm_password: confirmPassword,
      });
      setDone(true);
    } catch (error) {
      if (axios.isAxiosError(error)) {
        const code = error.response?.data?.error;
        if (code === 'password_too_short') toast.error('Password must be at least 8 characters');
        else if (code === 'password_too_long') toast.error('Password must be at most 72 characters');
        else if (code === 'password_mismatch') toast.error('Passwords do not match');
        else toast.error(code || 'Unable to reset password');
      }
    } finally {
      setLoading(false);
    }
  }

  const loginHref = redirectUri
    ? `/?app_id=${encodeURIComponent(appId)}&redirect_uri=${encodeURIComponent(redirectUri)}`
    : `/?app_id=${encodeURIComponent(appId)}`;

  if (done) {
    return (
      <div className="space-y-4">
        <p className="text-sm" style={{ color: 'var(--muted)' }}>Password reset successfully.</p>
        <Link href={loginHref} className="text-sm underline underline-offset-4" style={{ color: 'var(--foreground)' }}>Return to login</Link>
      </div>
    );
  }

  return (
    <form className="space-y-4" onSubmit={handleSubmit}>
      <Input type="password" placeholder="New password" required minLength={8} maxLength={72} value={password} onChange={(event) => setPassword(event.target.value)} />
      <Input type="password" placeholder="Confirm password" required minLength={8} maxLength={72} value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} />
      <Button type="submit" className="w-full justify-center" loading={loading}>Reset Password</Button>
    </form>
  );
}
