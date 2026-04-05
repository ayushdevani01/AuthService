'use client';

import Link from 'next/link';
import axios from 'axios';
import { useEffect, useMemo, useState } from 'react';
import toast from 'react-hot-toast';
import { Copy, KeyRound, Plus, ShieldCheck, Users } from 'lucide-react';
import { api } from '@/lib/api';
import { copyToClipboard, formatDate } from '@/lib/utils';
import type { AppRecord } from '@/lib/types';
import { Button, Card, ConfirmModal, Input, SectionHeading, Textarea, Toggle } from '@/components/ui';
import { EmptyState, StatCard } from '@/components/shell';

type CreateAppResponse = {
  app: AppRecord;
  api_key: string;
};

const initialCreateState = {
  name: '',
  redirect_urls: '',
  require_email_verification: false,
};

export function DashboardClient() {
  const [apps, setApps] = useState<AppRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState(initialCreateState);
  const [newApiKey, setNewApiKey] = useState<string | null>(null);
  const [showCreateConfirm, setShowCreateConfirm] = useState(false);

  useEffect(() => {
    void fetchApps();
  }, []);

  async function fetchApps() {
    setLoading(true);
    try {
      const { data } = await api.get('/api/v1/developers/apps');
      setApps(data.apps || []);
    } catch (error) {
      if (axios.isAxiosError(error)) {
        toast.error(error.response?.data?.error || 'Failed to load apps');
      }
    } finally {
      setLoading(false);
    }
  }

  function handleCreateIntent(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!form.name.trim()) {
      toast.error('App name is required');
      return;
    }
    setShowCreateConfirm(true);
  }

  async function handleCreate() {
    setCreating(true);
    try {
      const payload: Record<string, unknown> = { name: form.name.trim() };
      payload.require_email_verification = form.require_email_verification;
      const redirects = form.redirect_urls.split('\n').map((item) => item.trim()).filter(Boolean);
      if (redirects.length) payload.redirect_urls = redirects;

      const { data } = await api.post<CreateAppResponse>('/api/v1/developers/apps', payload);
      toast.success('Application created');
      setApps((current) => [data.app, ...current]);
      setNewApiKey(data.api_key);
      setForm(initialCreateState);
      setShowCreateConfirm(false);
    } catch (error) {
      if (axios.isAxiosError(error)) {
        toast.error(error.response?.data?.error || 'Failed to create app');
      }
    } finally {
      setCreating(false);
    }
  }

  const stats = useMemo(() => [
    { label: 'Applications', value: String(apps.length).padStart(2, '0'), sublabel: 'Connected developer apps', icon: <ShieldCheck className="h-4 w-4 text-foreground" /> },
    { label: 'OAuth Ready', value: String(apps.filter((app) => app.redirect_urls?.length).length).padStart(2, '0'), sublabel: 'Apps with redirect targets', icon: <KeyRound className="h-4 w-4 text-foreground" /> },
    { label: 'Identity Surface', value: apps.length ? 'Live' : 'Idle', sublabel: 'Provisioning status overview', icon: <Users className="h-4 w-4 text-foreground" /> },
  ], [apps]);

  return (
    <div className="space-y-6">
      <section className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
        <Card className="p-8">
          <SectionHeading eyebrow="Overview" title="Authentication infrastructure in one disciplined space" description="Create apps, issue credentials, and move through settings, providers, users, and keys without leaving the dashboard." />
          <div className="mt-8 grid gap-4 md:grid-cols-3">
            {stats.map((item) => <StatCard key={item.label} {...item} />)}
          </div>
        </Card>

        <Card className="p-8">
          <SectionHeading eyebrow="Create App" title="Provision a new application" description="The API key appears only once after creation, so save it immediately." />
          <form className="mt-6 space-y-4" onSubmit={handleCreateIntent}>
            <div>
              <label className="mb-2 block text-sm text-muted">App name</label>
              <Input value={form.name} onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))} placeholder="My New App" required />
            </div>
            <div>
              <label className="mb-2 block text-sm text-muted">Redirect URLs</label>
              <Textarea value={form.redirect_urls} onChange={(event) => setForm((current) => ({ ...current, redirect_urls: event.target.value }))} placeholder={"https://app.example.com/callback\nhttps://app.example.com/login"} />
            </div>
            <Toggle checked={form.require_email_verification} onChange={(checked) => setForm((current) => ({ ...current, require_email_verification: checked }))} label="Require verified email before login" />
            <Button type="submit" className="w-full justify-center"><Plus className="mr-2 h-4 w-4" />Create New App</Button>
          </form>
          {newApiKey ? (
            <div className="mt-6 rounded-3xl border border-[var(--border)] bg-app-panel p-5">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <p className="text-sm font-medium text-foreground">Save this API key</p>
                  <p className="mt-1 text-sm text-muted">It will not be shown again.</p>
                </div>
                <Button variant="secondary" onClick={() => copyToClipboard(newApiKey).then(() => toast.success('API key copied'))}><Copy className="mr-2 h-4 w-4" />Copy</Button>
              </div>
              <pre className="mt-4 overflow-x-auto rounded-2xl border border-[var(--border)] bg-[var(--background-alt)] p-4 text-xs text-muted">{newApiKey}</pre>
            </div>
          ) : null}
        </Card>
      </section>

      <section className="space-y-4">
        <SectionHeading eyebrow="Apps" title="Your application portfolio" description="Open an app to manage settings, OAuth providers, keys, and end-user accounts." />
        {loading ? (
          <Card className="p-10 text-sm text-muted">Loading applications...</Card>
        ) : apps.length === 0 ? (
          <EmptyState title="No applications yet" description="Create your first app to receive an `app_id`, API key, signing key, and hosted login flow." />
        ) : (
          <div className="grid gap-4 xl:grid-cols-2">
            {apps.map((app) => (
              <Link href={`/dashboard/apps/${app.app_id}`} key={app.app_id} className="luxury-card p-6 transition hover:border-[var(--border-strong)]">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <p className="text-lg font-medium text-foreground">{app.name}</p>
                    <p className="mt-1 text-sm text-muted">Created {formatDate(app.created_at)}</p>
                  </div>
                  <Button type="button" variant="ghost" className="shrink-0" onClick={(event) => { event.preventDefault(); void copyToClipboard(app.app_id).then(() => toast.success('App ID copied')); }}><Copy className="mr-2 h-4 w-4" />Copy ID</Button>
                </div>
                <div className="mt-6 grid gap-3 md:grid-cols-2">
                  <div className="rounded-2xl border border-[var(--border)] bg-[var(--background-alt)] p-4">
                    <p className="text-xs uppercase tracking-[0.3em] text-muted">App ID</p>
                    <p className="mt-3 break-all text-sm text-foreground">{app.app_id}</p>
                  </div>
                  <div className="rounded-2xl border border-[var(--border)] bg-[var(--background-alt)] p-4">
                    <p className="text-xs uppercase tracking-[0.3em] text-muted">Redirect URLs</p>
                    <p className="mt-3 text-sm text-foreground">{app.redirect_urls?.length || 0} configured</p>
                  </div>
                </div>
              </Link>
            ))}
          </div>
        )}
      </section>

      <ConfirmModal open={showCreateConfirm} title="Create this application?" description="This will provision a new app and reveal a one-time API key. Confirm to continue." confirmLabel="Create App" loading={creating} onCancel={() => setShowCreateConfirm(false)} onConfirm={handleCreate} />
    </div>
  );
}
