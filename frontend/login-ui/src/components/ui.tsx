'use client';

import { Loader2 } from 'lucide-react';
import type { ReactNode } from 'react';

export function Button({ children, loading, variant = 'primary', className = '', ...props }: React.ButtonHTMLAttributes<HTMLButtonElement> & { loading?: boolean; variant?: 'primary' | 'secondary' }) {
  const variantClass = variant === 'primary' ? 'button-primary' : 'button-secondary';
  return <button className={`${variantClass} ${className}`} disabled={loading || props.disabled} {...props}>{loading ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}{children}</button>;
}

export function Input(props: React.InputHTMLAttributes<HTMLInputElement>) {
  return <input className={`input-shell ${props.className || ''}`} {...props} />;
}

export function Panel({ children, className = '' }: { children: ReactNode; className?: string }) {
  return <div className={`glass-panel rounded-xl2 p-8 ${className}`}>{children}</div>;
}

export function Toggle({ checked, onChange, label }: { checked: boolean; onChange: (value: boolean) => void; label: string }) {
  return (
    <label className="inline-flex items-center gap-3 text-sm" style={{ color: 'var(--muted)' }}>
      <button type="button" role="switch" aria-checked={checked} onClick={() => onChange(!checked)} className={`toggle-shell ${checked ? 'toggle-shell-active' : ''}`}>
        <span className={`toggle-knob ${checked ? 'translate-x-5' : ''}`} />
      </button>
      <span>{label}</span>
    </label>
  );
}
