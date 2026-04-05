'use client';

import axios from 'axios';
import { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import toast from 'react-hot-toast';
import { Copy, ExternalLink, RotateCcw, Search, Trash2 } from 'lucide-react';
import { api } from '@/lib/api';
import { copyToClipboard, formatDate } from '@/lib/utils';
import type { AppRecord, AppUser, OAuthProvider, PaginatedUsers, SigningKey } from '@/lib/types';
import { Button, Card, ConfirmModal, Input, PillMultiSelect, SectionHeading, Textarea, Toggle } from '@/components/ui';

type Props = { appId: string };
type TabKey = 'overview' | 'settings' | 'providers' | 'users' | 'keys';
type ConfirmState = { type: 'delete-app' | 'delete-provider' | 'rotate-api-key' | 'save-settings' | 'save-provider'; providerName?: string } | null;

const tabs: TabKey[] = ['overview', 'settings', 'providers', 'users', 'keys'];

const scopeOptions = {
  google: ['openid', 'profile', 'email'],
  github: ['read:user', 'user:email'],
} as const;

const providerScopeDefaults = {
  google: ['openid', 'email', 'profile'],
  github: ['read:user', 'user:email'],
} as const;

export function AppDetailClient({ appId }: Props) {
  const router = useRouter();
  const [app, setApp] = useState<AppRecord | null>(null);
  const [activeTab, setActiveTab] = useState<TabKey>('overview');
  const [loading, setLoading] = useState(true);
  const [settingsForm, setSettingsForm] = useState({ name: '', redirect_urls: '' });
  const [savingSettings, setSavingSettings] = useState(false);
  const [providers, setProviders] = useState<OAuthProvider[]>([]);
  const [providerForm, setProviderForm] = useState<{ provider: 'google' | 'github'; client_id: string; client_secret: string; scopes: string[]; enabled: boolean }>({
    provider: 'google',
    client_id: '',
    client_secret: '',
    scopes: [...providerScopeDefaults.google],
    enabled: true,
  });
  const [providerLoading, setProviderLoading] = useState(false);
  const [users, setUsers] = useState<AppUser[]>([]);
  const [userSearch, setUserSearch] = useState('');
  const [providerFilter, setProviderFilter] = useState('');
  const [pageToken, setPageToken] = useState('');
  const [pageHistory, setPageHistory] = useState<string[]>([]);
  const [nextPageToken, setNextPageToken] = useState('');
  const [totalCount, setTotalCount] = useState(0);
  const [keys, setKeys] = useState<SigningKey[]>([]);
  const [includeExpired, setIncludeExpired] = useState(false);
  const [gracePeriodHours, setGracePeriodHours] = useState('24');
  const [revealedApiKey, setRevealedApiKey] = useState<string | null>(null);
  const [confirmState, setConfirmState] = useState<ConfirmState>(null);
  const [confirmLoading, setConfirmLoading] = useState(false);

  useEffect(() => {
    void fetchApp();
  }, [appId]);

  useEffect(() => {
    if (activeTab === 'providers') void fetchProviders();
    if (activeTab === 'users') void fetchUsers(pageToken);
    if (activeTab === 'keys') void fetchKeys();
  }, [activeTab, includeExpired]);

  async function fetchApp() {
    setLoading(true);
    try {
      const { data } = await api.get(`/api/v1/developers/apps/${appId}`);
      setApp(data.app);
      setSettingsForm({
        name: data.app.name || '',
        redirect_urls: (data.app.redirect_urls || []).join('\n'),
      });
    } catch (error) {
      if (axios.isAxiosError(error)) toast.error(error.response?.data?.error || 'Failed to load app');
    } finally {
      setLoading(false);
    }
  }

  async function fetchProviders() {
    try {
      const { data } = await api.get(`/api/v1/developers/apps/${appId}/providers`);
      setProviders(data.providers || []);
    } catch (error) {
      if (axios.isAxiosError(error)) toast.error(error.response?.data?.error || 'Failed to load providers');
    }
  }

  async function fetchUsers(token = '') {
    try {
      const params = new URLSearchParams();
      params.set('page_size', '20');
      if (token) params.set('page_token', token);
      if (providerFilter) params.set('provider_filter', providerFilter);
      if (userSearch.trim()) params.set('email_search', userSearch.trim());
      const { data } = await api.get<PaginatedUsers>(`/api/v1/developers/apps/${appId}/users?${params.toString()}`);
      setUsers(data.users || []);
      setNextPageToken(data.next_page_token || '');
      setTotalCount(data.total_count || 0);
    } catch (error) {
      if (axios.isAxiosError(error)) toast.error(error.response?.data?.error || 'Failed to load users');
    }
  }

  async function fetchKeys() {
    try {
      const { data } = await api.get(`/api/v1/developers/apps/${appId}/keys?include_expired=${includeExpired}`);
      const keyList = (data.keys || []) as SigningKey[];
      setKeys(includeExpired ? keyList : keyList.filter((key) => key.is_active || !key.expires_at));
    } catch (error) {
      if (axios.isAxiosError(error)) toast.error(error.response?.data?.error || 'Failed to load keys');
    }
  }

  function requestSaveSettings() {
    setConfirmState({ type: 'save-settings' });
  }

  async function saveSettings() {
    if (!app) return;
    setSavingSettings(true);
    try {
      const payload: Record<string, unknown> = {};
      if (settingsForm.name.trim() !== app.name) payload.name = settingsForm.name.trim();
      const currentRedirects = (app.redirect_urls || []).join('\n');
      if (settingsForm.redirect_urls !== currentRedirects) {
        payload.redirect_urls = settingsForm.redirect_urls.split('\n').map((item) => item.trim()).filter(Boolean);
      }
      const { data } = await api.patch(`/api/v1/developers/apps/${appId}`, payload);
      setApp(data.app);
      setSettingsForm({ name: data.app.name || '', redirect_urls: (data.app.redirect_urls || []).join('\n') });
      toast.success('App updated');
    } catch (error) {
      if (axios.isAxiosError(error)) toast.error(error.response?.data?.error || 'Failed to update app');
    } finally {
      setSavingSettings(false);
      setConfirmState(null);
    }
  }

  async function deleteApp() {
    setConfirmLoading(true);
    try {
      await api.delete(`/api/v1/developers/apps/${appId}`);
      toast.success('App deleted');
      router.push('/dashboard');
    } catch (error) {
      if (axios.isAxiosError(error)) toast.error(error.response?.data?.error || 'Failed to delete app');
    } finally {
      setConfirmLoading(false);
      setConfirmState(null);
    }
  }

  function requestSaveProvider(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!providerForm.client_id.trim() || !providerForm.client_secret.trim()) {
      toast.error('Client ID and client secret are required');
      return;
    }
    if (providerForm.scopes.length === 0) {
      toast.error('Select at least one scope');
      return;
    }
    setConfirmState({ type: 'save-provider', providerName: providerForm.provider });
  }

  async function createOrUpdateProvider() {
    setProviderLoading(true);
    try {
      const existing = providers.find((provider) => provider.provider === providerForm.provider);
      const payload = {
        client_id: providerForm.client_id,
        client_secret: providerForm.client_secret,
        scopes: providerForm.scopes,
        enabled: providerForm.enabled,
      };

      if (existing) {
        await api.patch(`/api/v1/developers/apps/${appId}/providers/${providerForm.provider}`, payload);
        toast.success('Provider updated');
      } else {
        await api.post(`/api/v1/developers/apps/${appId}/providers`, { provider: providerForm.provider, ...payload });
        toast.success('Provider added');
      }

      setProviderForm({
        provider: 'google',
        client_id: '',
        client_secret: '',
        scopes: [...providerScopeDefaults.google],
        enabled: true,
      });
      setConfirmState(null);
      await fetchProviders();
    } catch (error) {
      if (axios.isAxiosError(error)) toast.error(error.response?.data?.error || 'Failed to save provider');
    } finally {
      setProviderLoading(false);
    }
  }

  async function toggleProvider(provider: OAuthProvider) {
    try {
      await api.patch(`/api/v1/developers/apps/${appId}/providers/${provider.provider}`, { enabled: !provider.enabled, scopes: provider.scopes });
      toast.success('Provider updated');
      await fetchProviders();
    } catch (error) {
      if (axios.isAxiosError(error)) toast.error(error.response?.data?.error || 'Failed to update provider');
    }
  }

  async function removeProvider(providerName: string) {
    setConfirmLoading(true);
    try {
      await api.delete(`/api/v1/developers/apps/${appId}/providers/${providerName}`);
      toast.success('Provider deleted');
      await fetchProviders();
    } catch (error) {
      if (axios.isAxiosError(error)) toast.error(error.response?.data?.error || 'Failed to delete provider');
    } finally {
      setConfirmLoading(false);
      setConfirmState(null);
    }
  }

  async function rotateApiKey() {
    setConfirmLoading(true);
    try {
      const { data } = await api.post(`/api/v1/developers/apps/${appId}/rotate-secret`);
      setRevealedApiKey(data.new_api_key);
      toast.success('API key rotated');
    } catch (error) {
      if (axios.isAxiosError(error)) toast.error(error.response?.data?.error || 'Failed to rotate API key');
    } finally {
      setConfirmLoading(false);
      setConfirmState(null);
    }
  }

  function handleConfirmAction() {
    if (!confirmState) return;
    if (confirmState.type === 'delete-app') return void deleteApp();
    if (confirmState.type === 'rotate-api-key') return void rotateApiKey();
    if (confirmState.type === 'save-settings') return void saveSettings();
    if (confirmState.type === 'save-provider') return void createOrUpdateProvider();
    if (confirmState.type === 'delete-provider' && confirmState.providerName) return void removeProvider(confirmState.providerName);
  }

  async function rotateKeys() {
    try {
      const { data } = await api.post(`/api/v1/developers/apps/${appId}/rotate-keys`, { grace_period_hours: Number(gracePeriodHours) || 24 });
      toast.success(`New key ${data.new_key?.kid || 'created'}`);
      if (data.old_key?.kid) toast.success(`Old key ${data.old_key.kid} is in grace period`);
      await fetchKeys();
    } catch (error) {
      if (axios.isAxiosError(error)) toast.error(error.response?.data?.error || 'Failed to rotate signing keys');
    }
  }

  const hostedLoginUrl = useMemo(() => {
    if (!app?.app_id) return '';
    const preferredRedirect = app.redirect_urls?.[0] || 'http://localhost:3000/auth/callback';
    return `http://localhost:3001/?app_id=${encodeURIComponent(app.app_id)}&redirect_uri=${encodeURIComponent(preferredRedirect)}`;
  }, [app?.app_id, app?.redirect_urls]);

  if (loading) return <Card className="p-10 text-sm text-muted">Loading application...</Card>;
  if (!app) return <Card className="p-10 text-sm text-muted">App not found.</Card>;

  return (
    <div className="space-y-6">
      <Card className="p-8">
        <SectionHeading eyebrow="Application" title={app.name} description="Move between overview, settings, providers, users, and cryptographic materials." />
        <div className="mt-6 flex flex-wrap gap-3">
          {tabs.map((tab) => (
            <Button key={tab} variant={activeTab === tab ? 'primary' : 'secondary'} onClick={() => setActiveTab(tab)} className="capitalize">
              {tab}
            </Button>
          ))}
        </div>
      </Card>

      {activeTab === 'overview' ? (
        <div className="grid gap-6 xl:grid-cols-2">
          <Card className="space-y-5 p-8">
            <div>
              <p className="text-sm text-muted">Public App ID</p>
              <div className="mt-3 flex items-center gap-3">
                <p className="break-all text-lg text-foreground">{app.app_id}</p>
                <Button variant="secondary" onClick={() => copyToClipboard(app.app_id).then(() => toast.success('App ID copied'))}><Copy className="mr-2 h-4 w-4" />Copy</Button>
              </div>
            </div>
            <div>
              <p className="text-sm text-muted">Internal ID</p>
              <div className="mt-3 flex items-center gap-3">
                <p className="break-all text-sm text-foreground">{app.id}</p>
                <Button variant="secondary" onClick={() => copyToClipboard(app.id).then(() => toast.success('Internal ID copied'))}><Copy className="mr-2 h-4 w-4" />Copy</Button>
              </div>
            </div>
            <div className="grid gap-4 md:grid-cols-2">
              <div className="rounded-3xl border border-[var(--border)] bg-[var(--background-alt)] p-5">
                <p className="text-sm text-muted">Created</p>
                <p className="mt-2 text-foreground">{formatDate(app.created_at)}</p>
              </div>
              <div className="rounded-3xl border border-[var(--border)] bg-[var(--background-alt)] p-5">
                <p className="text-sm text-muted">Updated</p>
                <p className="mt-2 text-foreground">{formatDate(app.updated_at)}</p>
              </div>
            </div>
          </Card>

          <Card className="space-y-5 p-8">
            <div>
              <p className="text-sm text-muted">Hosted Login Preview URL</p>
              <p className="mt-3 break-all text-sm text-foreground">{hostedLoginUrl}</p>
            </div>
            <div className="flex flex-wrap gap-3">
              <Button variant="secondary" onClick={() => copyToClipboard(hostedLoginUrl).then(() => toast.success('Hosted login URL copied'))}><Copy className="mr-2 h-4 w-4" />Copy URL</Button>
              <a className="button-primary" href={hostedLoginUrl} target="_blank" rel="noreferrer"><ExternalLink className="mr-2 h-4 w-4" />Open Login UI</a>
            </div>
            <div>
              <p className="text-sm text-muted">Redirect URLs</p>
              <div className="mt-3 space-y-2">
                {(app.redirect_urls || []).map((url) => (
                  <div key={url} className="rounded-2xl border border-[var(--border)] bg-[var(--background-alt)] px-4 py-3 text-sm text-foreground">{url}</div>
                ))}
                {app.redirect_urls?.length === 0 ? <p className="text-sm text-muted">No redirect URLs configured.</p> : null}
              </div>
            </div>
          </Card>
        </div>
      ) : null}

      {activeTab === 'settings' ? (
        <div className="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
          <Card className="space-y-4 p-8">
            <SectionHeading eyebrow="Settings" title="Edit application metadata" description="Only changed fields are sent. Redirect URLs are sent as the full updated array when modified." />
            <div>
              <label className="mb-2 block text-sm text-muted">Name</label>
              <Input value={settingsForm.name} onChange={(event) => setSettingsForm((current) => ({ ...current, name: event.target.value }))} />
            </div>
            <div>
              <label className="mb-2 block text-sm text-muted">Redirect URLs</label>
              <Textarea value={settingsForm.redirect_urls} onChange={(event) => setSettingsForm((current) => ({ ...current, redirect_urls: event.target.value }))} />
            </div>
            <Button loading={savingSettings} onClick={requestSaveSettings}>Confirm Changes</Button>
          </Card>

          <Card className="space-y-4 p-8">
            <SectionHeading eyebrow="Danger Zone" title="Delete application" description="This permanently removes the app and returns you to the application list." />
            <Button variant="danger" onClick={() => setConfirmState({ type: 'delete-app' })}><Trash2 className="mr-2 h-4 w-4" />Delete App</Button>
          </Card>
        </div>
      ) : null}

      {activeTab === 'providers' ? (
        <div className="grid gap-6 xl:grid-cols-[1.05fr_0.95fr]">
          <Card className="space-y-4 p-8">
            <SectionHeading eyebrow="Providers" title="Configure Google and GitHub" description="Use curated scopes, explicit confirmations, and real provider defaults instead of raw free-text scope entry." />
            <form className="space-y-4" onSubmit={requestSaveProvider}>
              <div>
                <label className="mb-2 block text-sm text-muted">Provider</label>
                <select
                  className="input-shell"
                  value={providerForm.provider}
                  onChange={(event) => {
                    const provider = event.target.value as 'google' | 'github';
                    setProviderForm((current) => ({ ...current, provider, scopes: [...providerScopeDefaults[provider]] }));
                  }}
                >
                  <option value="google">Google</option>
                  <option value="github">GitHub</option>
                </select>
              </div>
              <div>
                <label className="mb-2 block text-sm text-muted">Client ID</label>
                <Input value={providerForm.client_id} onChange={(event) => setProviderForm((current) => ({ ...current, client_id: event.target.value }))} />
              </div>
              <div>
                <label className="mb-2 block text-sm text-muted">Client Secret</label>
                <Input value={providerForm.client_secret} onChange={(event) => setProviderForm((current) => ({ ...current, client_secret: event.target.value }))} type="password" />
              </div>
              <div className="space-y-3">
                <label className="block text-sm text-muted">Scopes</label>
                <PillMultiSelect options={[...scopeOptions[providerForm.provider]]} value={providerForm.scopes} onChange={(scopes) => setProviderForm((current) => ({ ...current, scopes }))} />
              </div>
              <Toggle checked={providerForm.enabled} onChange={(enabled) => setProviderForm((current) => ({ ...current, enabled }))} label="Enable provider after saving" />
              <Button type="submit" loading={providerLoading}>Confirm Provider Changes</Button>
            </form>
          </Card>

          <Card className="space-y-4 p-8">
            <SectionHeading eyebrow="Configured Providers" title="Existing OAuth integrations" description="Toggle providers on or off, or remove them completely." />
            <div className="space-y-3">
              {providers.map((provider) => (
                <div key={provider.id} className="rounded-3xl border border-[var(--border)] bg-[var(--background-alt)] p-5">
                  <div className="flex flex-wrap items-start justify-between gap-4">
                    <div>
                      <p className="text-base font-medium capitalize text-foreground">{provider.provider}</p>
                      <p className="mt-1 text-sm text-muted">Scopes: {provider.scopes.join(', ') || 'None'}</p>
                      <p className="mt-1 text-xs text-muted">Created {formatDate(provider.created_at)}</p>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      <Button variant="secondary" onClick={() => setProviderForm({ provider: provider.provider as 'google' | 'github', client_id: provider.client_id, client_secret: '', scopes: provider.scopes, enabled: provider.enabled })}>Edit</Button>
                      <Button variant="secondary" onClick={() => void toggleProvider(provider)}>{provider.enabled ? 'Disable' : 'Enable'}</Button>
                      <Button variant="danger" onClick={() => setConfirmState({ type: 'delete-provider', providerName: provider.provider })}>Delete</Button>
                    </div>
                  </div>
                </div>
              ))}
              {providers.length === 0 ? <p className="text-sm text-muted">No providers configured.</p> : null}
            </div>
          </Card>
        </div>
      ) : null}

      {activeTab === 'users' ? (
        <Card className="space-y-6 p-8">
          <div className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
            <SectionHeading eyebrow="Users" title="End-user accounts" description={`Showing ${users.length} records from ${totalCount} total users.`} />
            <div className="grid gap-3 md:grid-cols-3">
              <Input placeholder="Search email" value={userSearch} onChange={(event) => setUserSearch(event.target.value)} />
              <Input placeholder="Filter provider" value={providerFilter} onChange={(event) => setProviderFilter(event.target.value)} />
              <Button variant="secondary" onClick={() => { setPageToken(''); setPageHistory([]); void fetchUsers(''); }}><Search className="mr-2 h-4 w-4" />Apply Filters</Button>
            </div>
          </div>
          <div className="overflow-hidden rounded-3xl border border-[var(--border)]">
            <table className="min-w-full text-left text-sm">
              <thead className="bg-app-panel text-muted">
                <tr>
                  <th className="px-4 py-3">Name</th>
                  <th className="px-4 py-3">Email</th>
                  <th className="px-4 py-3">Provider</th>
                  <th className="px-4 py-3">Verified</th>
                  <th className="px-4 py-3">Created</th>
                </tr>
              </thead>
              <tbody>
                {users.map((user) => (
                  <tr key={user.id} className="border-t border-[var(--border)] bg-[var(--background-alt)] text-foreground">
                    <td className="px-4 py-3">{user.name || '—'}</td>
                    <td className="px-4 py-3">{user.email}</td>
                    <td className="px-4 py-3 capitalize">{user.provider || 'email'}</td>
                    <td className="px-4 py-3">{user.email_verified ? 'Yes' : 'No'}</td>
                    <td className="px-4 py-3">{formatDate(user.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            {users.length === 0 ? <p className="p-6 text-sm text-muted">No users found.</p> : null}
          </div>
          <div className="flex items-center justify-end gap-3">
            <Button variant="secondary" disabled={pageHistory.length === 0} onClick={() => { const history = [...pageHistory]; const previous = history.pop() || ''; setPageHistory(history); setPageToken(previous); void fetchUsers(previous); }}>Previous</Button>
            <Button variant="secondary" disabled={!nextPageToken} onClick={() => { setPageHistory((current) => [...current, pageToken]); setPageToken(nextPageToken); void fetchUsers(nextPageToken); }}>Next</Button>
          </div>
        </Card>
      ) : null}

      {activeTab === 'keys' ? (
        <div className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
          <Card className="space-y-6 p-8">
            <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
              <SectionHeading eyebrow="Signing Keys" title="Cryptographic material" description="Inspect current and rotated signing keys, with optional expired-key visibility." />
              <Toggle checked={includeExpired} onChange={setIncludeExpired} label="Include Expired" />
            </div>
            <div className="space-y-3">
              {keys.map((key) => (
                <div key={key.id} className={`rounded-3xl border p-5 ${key.is_active ? 'border-[var(--border-strong)] bg-app-panel' : 'border-[var(--border)] bg-[var(--background-alt)]'}`}>
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <p className="text-base font-medium text-foreground">{key.kid}</p>
                      <p className="mt-1 text-sm text-muted">Created {formatDate(key.created_at)}</p>
                    </div>
                    <span className={`rounded-full border px-3 py-1 text-xs ${key.is_active ? 'border-[var(--border-strong)] bg-[var(--accent)] text-[var(--accent-foreground)]' : 'border-[var(--border)] text-muted'}`}>{key.is_active ? 'Active' : 'Inactive'}</span>
                  </div>
                  <div className="mt-4 grid gap-3 text-sm text-muted md:grid-cols-2">
                    <p>Expires: {formatDate(key.expires_at)}</p>
                    <p>Rotated: {formatDate(key.rotated_at)}</p>
                  </div>
                  <pre className="mt-4 overflow-x-auto rounded-2xl border border-[var(--border)] bg-[var(--background-alt)] p-4 text-xs text-muted">{key.public_key}</pre>
                </div>
              ))}
              {keys.length === 0 ? <p className="text-sm text-muted">No keys returned.</p> : null}
            </div>
          </Card>

          <div className="space-y-6">
            <Card className="space-y-4 p-8">
              <SectionHeading eyebrow="API Secret" title="Rotate API key" description="The freshly generated API key is visible exactly once after rotation." />
              <Button onClick={() => setConfirmState({ type: 'rotate-api-key' })}><RotateCcw className="mr-2 h-4 w-4" />Rotate API Key</Button>
              {revealedApiKey ? (
                <div className="rounded-3xl border border-[var(--border)] bg-[var(--background-alt)] p-4">
                  <div className="flex items-center justify-between gap-4">
                    <p className="text-sm text-muted">New API key</p>
                    <Button variant="secondary" onClick={() => copyToClipboard(revealedApiKey).then(() => toast.success('API key copied'))}><Copy className="mr-2 h-4 w-4" />Copy</Button>
                  </div>
                  <pre className="mt-4 overflow-x-auto text-xs text-foreground">{revealedApiKey}</pre>
                </div>
              ) : null}
            </Card>

            <Card className="space-y-4 p-8">
              <SectionHeading eyebrow="Key Rotation" title="Rotate signing keys" description="Set a grace period in hours. The backend defaults to 24 if omitted or invalid." />
              <Input type="number" min="1" value={gracePeriodHours} onChange={(event) => setGracePeriodHours(event.target.value)} />
              <Button onClick={rotateKeys}><RotateCcw className="mr-2 h-4 w-4" />Rotate Signing Keys</Button>
            </Card>
          </div>
        </div>
      ) : null}

      <ConfirmModal
        open={Boolean(confirmState)}
        title={
          confirmState?.type === 'delete-app'
            ? 'Delete this application?'
            : confirmState?.type === 'delete-provider'
              ? `Delete ${confirmState.providerName} provider?`
              : confirmState?.type === 'save-settings'
                ? 'Save application changes?'
                : confirmState?.type === 'save-provider'
                  ? `Confirm ${confirmState.providerName} provider changes?`
                  : 'Rotate API key?'
        }
        description={
          confirmState?.type === 'delete-app'
            ? 'This permanently removes the application and its dashboard entry.'
            : confirmState?.type === 'delete-provider'
              ? 'This removes the OAuth provider configuration for this app.'
              : confirmState?.type === 'save-settings'
                ? 'This updates the application metadata and redirect configuration.'
                : confirmState?.type === 'save-provider'
                  ? 'This will create or update the selected provider using the chosen scopes.'
                  : 'A new API key will be generated and shown once. Save it immediately.'
        }
        confirmLabel={confirmState?.type === 'rotate-api-key' ? 'Rotate Key' : 'Confirm'}
        loading={confirmLoading || savingSettings || providerLoading}
        onCancel={() => setConfirmState(null)}
        onConfirm={handleConfirmAction}
      />
    </div>
  );
}
