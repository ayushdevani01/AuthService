'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { AppWindow, BookOpenText, ChevronRight, Compass, KeyRound, Layers3, LogOut, Shield, Users } from 'lucide-react';
import toast from 'react-hot-toast';
import { useAuthStore } from '@/store/auth';
import { api } from '@/lib/api';
import { Button, ThemeToggle } from '@/components/ui';
import { cn } from '@/lib/utils';
import type { ReactNode } from 'react';

const links = [
  { href: '/dashboard', label: 'Applications', icon: Layers3 },
  { href: '/dashboard/profile', label: 'Profile', icon: Shield },
  { href: '/', label: 'Marketing', icon: Compass },
  { href: '/#docs', label: 'Docs', icon: BookOpenText },
];

export function DashboardShell({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const developer = useAuthStore((state) => state.developer);
  const logoutLocal = useAuthStore((state) => state.logoutLocal);

  async function handleLogout() {
    try {
      await api.post('/api/v1/developers/logout', {});
    } catch {
    } finally {
      logoutLocal();
      toast.success('Logged out');
      router.push('/login');
    }
  }

  return (
    <div className="min-h-screen">
      <div className="mx-auto flex min-h-screen max-w-[1600px] gap-6 px-4 py-4 lg:px-6">
        <aside className="glass-panel hidden w-80 shrink-0 rounded-[2rem] p-6 lg:flex lg:flex-col">
          <div className="mb-10 flex items-center gap-4">
            <div className="flex h-14 w-14 items-center justify-center rounded-3xl border border-[var(--border)] bg-app-panel">
              <AppWindow className="h-6 w-6" />
            </div>
            <div>
              <p className="text-xs uppercase tracking-[0.35em] text-muted">AuthService</p>
              <h1 className="text-lg font-semibold text-foreground">Developer Console</h1>
            </div>
          </div>

          <div className="mb-6 flex items-center justify-between gap-3 rounded-2xl border border-[var(--border)] bg-app-panel p-3">
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-muted">Theme</p>
              <p className="text-sm text-foreground">Global site mode</p>
            </div>
            <ThemeToggle />
          </div>

          <nav className="space-y-2">
            {links.map((link) => {
              const Icon = link.icon;
              const active = link.href.startsWith('/#') ? false : pathname === link.href || pathname.startsWith(`${link.href}/`);
              return (
                <Link
                  key={link.href}
                  href={link.href}
                  className={cn(
                    'flex items-center justify-between rounded-2xl px-4 py-3 text-sm transition',
                    active ? 'bg-[var(--accent)] text-[var(--accent-foreground)]' : 'text-muted hover:bg-app-panel hover:text-foreground',
                  )}
                >
                  <span className="flex items-center gap-3">
                    <Icon className="h-4 w-4" />
                    {link.label}
                  </span>
                  <ChevronRight className="h-4 w-4" />
                </Link>
              );
            })}
          </nav>

          <div className="mt-10 rounded-3xl border border-[var(--border)] bg-app-panel p-5">
            <p className="text-xs uppercase tracking-[0.35em] text-muted">Signed In</p>
            <p className="mt-3 text-lg font-medium text-foreground">{developer?.name || 'Developer'}</p>
            <p className="mt-1 text-sm text-muted">{developer?.email}</p>
          </div>

          <div className="mt-auto space-y-3 rounded-3xl border border-[var(--border)] bg-app-panel p-5">
            <div className="flex items-start gap-3 text-sm text-muted">
              <KeyRound className="mt-0.5 h-4 w-4 text-foreground" />
              Manage apps, providers, keys, and users in a premium, documentation-friendly control surface.
            </div>
            <Button variant="secondary" onClick={handleLogout} className="w-full justify-center">
              <LogOut className="mr-2 h-4 w-4" />
              Logout
            </Button>
          </div>
        </aside>

        <div className="flex min-h-screen flex-1 flex-col gap-6 pb-10">
          <header className="glass-panel rounded-[2rem] p-5 lg:hidden">
            <div className="flex items-center justify-between gap-4">
              <div>
                <p className="text-xs uppercase tracking-[0.35em] text-muted">AuthService</p>
                <h1 className="text-xl font-semibold text-foreground">Developer Console</h1>
              </div>
              <ThemeToggle />
            </div>
          </header>
          {children}
        </div>
      </div>
    </div>
  );
}

export function StatCard({ label, value, sublabel, icon }: { label: string; value: string; sublabel?: string; icon?: ReactNode }) {
  return (
    <div className="luxury-card p-6">
      <div className="flex items-center justify-between">
        <span className="text-sm text-muted">{label}</span>
        {icon}
      </div>
      <p className="mt-6 text-3xl font-semibold tracking-tight text-foreground">{value}</p>
      {sublabel ? <p className="mt-2 text-sm text-muted">{sublabel}</p> : null}
    </div>
  );
}

export function EmptyState({ title, description, icon }: { title: string; description: string; icon?: ReactNode }) {
  return (
    <div className="luxury-card flex min-h-72 flex-col items-center justify-center p-8 text-center">
      <div className="mb-4 rounded-3xl border border-[var(--border)] bg-app-panel p-4">{icon ?? <Users className="h-7 w-7 text-foreground" />}</div>
      <h3 className="text-xl font-medium text-foreground">{title}</h3>
      <p className="mt-2 max-w-md text-sm text-muted">{description}</p>
    </div>
  );
}
