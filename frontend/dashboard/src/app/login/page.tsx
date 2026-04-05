'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import axios from 'axios';
import { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { api } from '@/lib/api';
import { Button, Card, Input, SectionHeading, ThemeToggle } from '@/components/ui';
import { useAuthStore } from '@/store/auth';
import type { Developer } from '@/lib/types';

export default function LoginPage() {
  const router = useRouter();
  const setAuth = useAuthStore((state) => state.setAuth);
  const accessToken = useAuthStore((state) => state.accessToken);
  const hydrated = useAuthStore((state) => state.hydrated);
  const [form, setForm] = useState({ email: '', password: '' });
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (hydrated && accessToken) {
      router.replace('/dashboard');
    }
  }, [accessToken, hydrated, router]);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    try {
      const { data } = await api.post('/api/v1/developers/login', form);
      setAuth({ accessToken: data.access_token, developer: data.developer as Developer });
      toast.success('Welcome back');
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
      <div className="grid w-full max-w-6xl gap-6 lg:grid-cols-[1.15fr_0.85fr]">
        <Card className="flex min-h-[680px] flex-col justify-between overflow-hidden p-10">
          <div>
            <div className="flex items-center justify-between gap-4">
              <p className="text-xs uppercase tracking-[0.45em] text-muted">AuthService</p>
              <ThemeToggle />
            </div>
            <h1 className="mt-6 max-w-2xl text-5xl font-semibold leading-tight text-foreground">A monochrome control room for authentication at scale.</h1>
            <p className="mt-6 max-w-xl text-base text-muted">Premium dark-first marketing cues with light-mode support, disciplined contrast, and precise developer tooling for apps, OAuth, keys, and users.</p>
          </div>
          <div className="grid gap-4 md:grid-cols-3">
            {[
              ['Applications', 'Issue app IDs and secrets with deliberate clarity.'],
              ['Providers', 'Shape OAuth experiences with confirm-first updates.'],
              ['Security', 'Rotate keys and inspect access with confidence.'],
            ].map(([title, copy]) => (
              <div key={title} className="rounded-3xl border border-[var(--border)] bg-app-panel p-5">
                <p className="text-sm font-medium text-foreground">{title}</p>
                <p className="mt-2 text-sm leading-6 text-muted">{copy}</p>
              </div>
            ))}
          </div>
        </Card>

        <Card className="p-8 lg:p-10">
          <SectionHeading eyebrow="Developer Login" title="Enter the dashboard" description="Sign in with your developer account to manage AuthService applications." />
          <form className="mt-10 space-y-4" onSubmit={handleSubmit}>
            <div>
              <label className="mb-2 block text-sm text-muted">Email</label>
              <Input type="email" required value={form.email} onChange={(event) => setForm((current) => ({ ...current, email: event.target.value }))} placeholder="dev@example.com" />
            </div>
            <div>
              <label className="mb-2 block text-sm text-muted">Password</label>
              <Input type="password" required value={form.password} onChange={(event) => setForm((current) => ({ ...current, password: event.target.value }))} placeholder="Enter your password" />
            </div>
            <Button type="submit" loading={loading} className="mt-4 w-full justify-center">Sign In</Button>
          </form>
          <p className="mt-6 text-sm text-muted">Need an account? <Link href="/register" className="text-foreground underline underline-offset-4">Create one</Link></p>
        </Card>
      </div>
    </main>
  );
}
