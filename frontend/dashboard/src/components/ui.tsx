'use client';

import { Check, Copy, Loader2, Monitor, Moon, Sun } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useThemeStore } from '@/store/theme';
import type { ReactNode } from 'react';
import { useState } from 'react';

export function Button({
  children,
  className,
  variant = 'primary',
  loading,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
  loading?: boolean;
}) {
  return (
    <button
      className={cn(
        variant === 'primary' && 'button-primary',
        variant === 'secondary' && 'button-secondary',
        variant === 'ghost' && 'button-ghost',
        variant === 'danger' && 'button-danger',
        className,
      )}
      disabled={loading || props.disabled}
      {...props}
    >
      {loading ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
      {children}
    </button>
  );
}

export function Input(props: React.InputHTMLAttributes<HTMLInputElement>) {
  return <input className={cn('input-shell', props.className)} {...props} />;
}

export function Textarea(props: React.TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className={cn('input-shell min-h-28 resize-y', props.className)} {...props} />;
}

export function Card({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn('luxury-card p-6', className)}>{children}</div>;
}

export function SectionHeading({ eyebrow, title, description }: { eyebrow?: string; title: string; description?: string }) {
  return (
    <div className="space-y-2">
      {eyebrow ? <p className="text-xs uppercase tracking-[0.35em] text-muted">{eyebrow}</p> : null}
      <h2 className="text-2xl font-semibold tracking-tight text-foreground">{title}</h2>
      {description ? <p className="max-w-2xl text-sm text-muted">{description}</p> : null}
    </div>
  );
}

export function ThemeToggle() {
  const mode = useThemeStore((state) => state.mode);
  const toggleMode = useThemeStore((state) => state.toggleMode);

  return (
    <button type="button" onClick={toggleMode} className="theme-toggle" aria-label={`Switch to ${mode === 'dark' ? 'light' : 'dark'} mode`}>
      {mode === 'dark' ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
      <span>{mode === 'dark' ? 'Light Mode' : 'Dark Mode'}</span>
      <Monitor className="h-4 w-4 opacity-60" />
    </button>
  );
}

export function Toggle({ checked, onChange, label }: { checked: boolean; onChange: (value: boolean) => void; label: string }) {
  return (
    <label className="inline-flex items-center gap-3 text-sm text-muted">
      <button type="button" role="switch" aria-checked={checked} onClick={() => onChange(!checked)} className={cn('toggle-shell', checked && 'toggle-shell-active')}>
        <span className={cn('toggle-knob', checked && 'translate-x-5')} />
      </button>
      <span>{label}</span>
    </label>
  );
}

export function PillMultiSelect({
  options,
  value,
  onChange,
}: {
  options: string[];
  value: string[];
  onChange: (value: string[]) => void;
}) {
  return (
    <div className="flex flex-wrap gap-2">
      {options.map((option) => {
        const active = value.includes(option);
        return (
          <button
            key={option}
            type="button"
            className={cn('scope-pill', active && 'scope-pill-active')}
            onClick={() => onChange(active ? value.filter((item) => item !== option) : [...value, option])}
          >
            <span className={cn('scope-pill-check', active && 'scope-pill-check-active')}>
              <Check className="h-3.5 w-3.5" />
            </span>
            {option}
          </button>
        );
      })}
    </div>
  );
}

export function ConfirmModal({
  open,
  title,
  description,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  loading,
  onConfirm,
  onCancel,
}: {
  open: boolean;
  title: string;
  description: string;
  confirmLabel?: string;
  cancelLabel?: string;
  loading?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-4 backdrop-blur-md">
      <div className="luxury-card w-full max-w-md p-6">
        <div className="space-y-2">
          <h3 className="text-xl font-semibold text-foreground">{title}</h3>
          <p className="text-sm leading-6 text-muted">{description}</p>
        </div>
        <div className="mt-6 flex justify-end gap-3">
          <Button variant="secondary" onClick={onCancel} disabled={loading}>{cancelLabel}</Button>
          <Button onClick={onConfirm} loading={loading}>{confirmLabel}</Button>
        </div>
      </div>
    </div>
  );
}

export function CodeBlock({ title, code }: { title: string; code: string }) {
  const [copied, setCopied] = useState(false);

  async function handleCopy() {
    await navigator.clipboard.writeText(code);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1600);
  }

  return (
    <div className="rounded-3xl border border-[var(--border)] bg-[#0a0a0a] p-5 text-sm text-zinc-100">
      <div className="mb-3 flex items-center justify-between gap-3">
        <p className="text-xs uppercase tracking-[0.3em] text-zinc-400">{title}</p>
        <button type="button" onClick={handleCopy} className="inline-flex items-center gap-2 rounded-xl border border-white/10 px-3 py-1.5 text-xs text-zinc-300 transition hover:border-white/20 hover:bg-white/5 hover:text-white">
          {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
          {copied ? 'Copied' : 'Copy'}
        </button>
      </div>
      <pre className="overflow-x-auto whitespace-pre-wrap font-mono text-[13px] leading-6">{code}</pre>
    </div>
  );
}

export function CodeTabs({
  tabs,
  defaultTab,
}: {
  tabs: Array<{ key: string; label: string; code: string }>;
  defaultTab?: string;
}) {
  const [activeTab, setActiveTab] = useState(defaultTab || tabs[0]?.key || '');
  const current = tabs.find((tab) => tab.key === activeTab) || tabs[0];

  if (!current) return null;

  return (
    <div className="rounded-[1.75rem] border border-[var(--border)] bg-[var(--background-alt)] p-3">
      <div className="flex flex-wrap gap-2 border-b border-[var(--border)] px-2 pb-3">
        {tabs.map((tab) => (
          <button
            key={tab.key}
            type="button"
            onClick={() => setActiveTab(tab.key)}
            className={cn(
              'rounded-xl px-3 py-2 text-sm transition',
              activeTab === tab.key ? 'bg-[var(--accent)] text-[var(--accent-foreground)]' : 'text-muted hover:bg-[var(--panel)] hover:text-foreground',
            )}
          >
            {tab.label}
          </button>
        ))}
      </div>
      <div className="pt-3">
        <CodeBlock title={current.label} code={current.code} />
      </div>
    </div>
  );
}
