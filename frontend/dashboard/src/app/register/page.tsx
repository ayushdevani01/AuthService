'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import axios from 'axios';
import { useState } from 'react';
import toast from 'react-hot-toast';
import { api } from '@/lib/api';
import { Button, Card, Input, SectionHeading, ThemeToggle } from '@/components/ui';
import { useAuthStore } from '@/store/auth';
import type { Developer } from '@/lib/types';

export default function RegisterPage() {
  const router = useRouter();
  const setAuth = useAuthStore((state) => state.setAuth);
  const [form, setForm] = useState({ name: '', email: '', password: '' });
  const [loading, setLoading] = useState(false);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!form.name.trim()) {
      toast.error('Name is required');
      return;
    }
    if (form.password.length < 8) {
      toast.error('Password must be at least 8 characters');
      return;
    }

    setLoading(true);
    try {
      const { data } = await api.post('/api/v1/developers/register', {
        name: form.name.trim(),
        email: form.email,
        password: form.password,
      });
      setAuth({ accessToken: data.access_token, developer: data.developer as Developer });
      toast.success('Account created');
      router.push('/dashboard');
    } catch (error) {
      if (axios.isAxiosError(error)) {
        toast.error(error.response?.data?.error || 'Something went wrong');
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center px-4 py-12">
      <div className="w-full max-w-xl">
        <Card className="p-8 lg:p-10">
          <div className="mb-8 flex items-center justify-between gap-4">
            <SectionHeading eyebrow="Registration" title="Create your developer account" description="Join the AuthService dashboard with strict validation and a minimal monochrome workflow." />
            <ThemeToggle />
          </div>
          <form className="mt-10 space-y-4" onSubmit={handleSubmit}>
            <div>
              <label className="mb-2 block text-sm text-muted">Name</label>
              <Input required value={form.name} onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))} placeholder="John Doe" />
            </div>
            <div>
              <label className="mb-2 block text-sm text-muted">Email</label>
              <Input type="email" required value={form.email} onChange={(event) => setForm((current) => ({ ...current, email: event.target.value }))} placeholder="dev@example.com" />
            </div>
            <div>
              <label className="mb-2 block text-sm text-muted">Password</label>
              <Input type="password" required value={form.password} onChange={(event) => setForm((current) => ({ ...current, password: event.target.value }))} placeholder="Minimum 8 characters" />
            </div>
            <Button type="submit" loading={loading} className="w-full justify-center">Create Account</Button>
          </form>
          <p className="mt-6 text-sm text-muted">Already have an account? <Link href="/login" className="text-foreground underline underline-offset-4">Sign in</Link></p>
        </Card>
      </div>
    </main>
  );
}
