'use client';

import Link from 'next/link';
import axios from 'axios';
import { useState } from 'react';
import toast from 'react-hot-toast';
import { api } from '@/lib/api';
import { Button, Input } from '@/components/ui';

export function ResetPasswordForm({ appId, token }: { appId: string; token: string }) {
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
    setLoading(true);
    try {
      await api.post('/auth/reset-password', { app_id: appId, token, new_password: password });
      setDone(true);
    } catch (error) {
      if (axios.isAxiosError(error)) toast.error(error.response?.data?.error || 'Unable to reset password');
    } finally {
      setLoading(false);
    }
  }

  if (done) return <div className="space-y-4"><p className="text-sm" style={{ color: 'var(--muted)' }}>Password reset successfully.</p><Link href={`/?app_id=${encodeURIComponent(appId)}`} className="text-sm underline underline-offset-4" style={{ color: 'var(--foreground)' }}>Return to login</Link></div>;
  return <form className="space-y-4" onSubmit={handleSubmit}><Input type="password" placeholder="New password" required value={password} onChange={(event) => setPassword(event.target.value)} /><Input type="password" placeholder="Confirm password" required value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} /><Button type="submit" className="w-full justify-center" loading={loading}>Reset Password</Button></form>;
}
