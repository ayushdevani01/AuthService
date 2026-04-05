'use client';

import axios from 'axios';
import { useEffect, useState } from 'react';
import toast from 'react-hot-toast';
import { api } from '@/lib/api';
import { formatDate } from '@/lib/utils';
import { useAuthStore } from '@/store/auth';
import { Button, Card, Input, SectionHeading } from '@/components/ui';
import type { Developer } from '@/lib/types';

export function ProfileClient() {
  const developer = useAuthStore((state) => state.developer);
  const updateDeveloper = useAuthStore((state) => state.updateDeveloper);
  const [name, setName] = useState(developer?.name ?? '');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setName(developer?.name ?? '');
  }, [developer?.name]);

  async function handleSave(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    try {
      const payload: Record<string, string> = {};
      if (name.trim() && name.trim() !== developer?.name) payload.name = name.trim();
      if (password) payload.password = password;
      const { data } = await api.patch('/api/v1/developers/profile', payload);
      updateDeveloper(data.developer as Developer);
      setPassword('');
      toast.success('Profile updated');
    } catch (error) {
      if (axios.isAxiosError(error)) {
        toast.error(error.response?.data?.error || 'Failed to update profile');
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-6">
      <Card className="p-8">
        <SectionHeading eyebrow="Profile" title="Developer identity" description="Review account metadata and update your name or password when needed." />
      </Card>
      <div className="grid gap-6 xl:grid-cols-[0.9fr_1.1fr]">
        <Card className="p-8">
          {!developer ? (
            <p className="text-sm text-zinc-500">Profile data unavailable. Please sign in again.</p>
          ) : (
            <div className="space-y-5 text-sm">
              <div>
                <p className="text-zinc-500">Email</p>
                <p className="mt-2 text-white">{developer?.email}</p>
              </div>
              <div>
                <p className="text-zinc-500">Created</p>
                <p className="mt-2 text-white">{formatDate(developer?.created_at)}</p>
              </div>
              <div>
                <p className="text-zinc-500">Updated</p>
                <p className="mt-2 text-white">{formatDate(developer?.updated_at)}</p>
              </div>
            </div>
          )}
        </Card>

        <Card className="p-8">
          <form className="space-y-4" onSubmit={handleSave}>
            <div>
              <label className="mb-2 block text-sm text-zinc-400">Name</label>
              <Input value={name} onChange={(event) => setName(event.target.value)} required />
            </div>
            <div>
              <label className="mb-2 block text-sm text-zinc-400">New Password</label>
              <Input type="password" value={password} onChange={(event) => setPassword(event.target.value)} placeholder="Leave blank to keep current password" />
            </div>
            <Button type="submit" loading={loading}>Save Changes</Button>
          </form>
        </Card>
      </div>
    </div>
  );
}
