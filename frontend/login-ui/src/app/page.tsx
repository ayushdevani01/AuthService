'use client';

import { Suspense } from 'react';
import Link from 'next/link';
import { LockKeyhole, ShieldCheck, Sparkles } from 'lucide-react';
import { useQueryParams } from '@/hooks/useQueryParams';
import { usePublicAppConfig } from '@/hooks/usePublicAppConfig';
import { Panel } from '@/components/ui';
import { ThemeToggle } from '@/components/theme-toggle';
import { LoginForm } from '@/components/LoginForm';
import { RegisterForm } from '@/components/RegisterForm';
import { ForgotPasswordForm } from '@/components/ForgotPasswordForm';
import { ResetPasswordForm } from '@/components/ResetPasswordForm';
import { VerifyEmailView } from '@/components/VerifyEmailView';

const trustPoints = [
  [LockKeyhole, 'Password + OAuth', 'Support email/password, Google, and GitHub from one hosted authentication surface.'],
  [ShieldCheck, 'Session-ready', 'After success, tokens are returned to your callback route so your app can establish the session immediately.'],
  [Sparkles, 'Developer-owned', 'Use your own app ID, redirect URI, and provider credentials without rebuilding auth UI for every product.'],
] as const;

function EmailDisabledNotice({ appId, redirectUri }: { appId: string; redirectUri: string }) {
  const loginHref = redirectUri
    ? `/?app_id=${encodeURIComponent(appId)}&redirect_uri=${encodeURIComponent(redirectUri)}`
    : `/?app_id=${encodeURIComponent(appId)}`;

  return (
    <div className="space-y-4">
      <p className="text-sm" style={{ color: 'var(--muted)' }}>Email sign-in is not enabled for this app.</p>
      <Link href={loginHref} className="text-sm underline underline-offset-4" style={{ color: 'var(--foreground)' }}>
        Back to sign in
      </Link>
    </div>
  );
}

function HostedLoginInner() {
  const { appId, redirectUri, mode, token } = useQueryParams();
  const configState = usePublicAppConfig(appId);
  const titleMap = {
    login: 'Sign in to continue',
    register: 'Create your account',
    forgot: 'Recover access',
    reset: 'Choose a new password',
    verify: 'Confirm your email',
  } as const;

  const emailEnabled = configState.status === 'ready' ? configState.config.auth_methods.email : true;
  const authMethods = configState.status === 'ready'
    ? configState.config.auth_methods
    : { email: true, google: false, github: false };

  const emailModesBlocked = !emailEnabled && (mode === 'register' || mode === 'forgot' || mode === 'reset');
  const missingParams = !appId || (mode !== 'forgot' && mode !== 'verify' && mode !== 'reset' && !redirectUri);

  return (
    <main className="flex min-h-screen items-center justify-center px-4 py-12">
      <div className="grid w-full max-w-6xl gap-6 lg:grid-cols-[1.05fr_0.95fr]">
        <Panel>
          <div className="flex items-center justify-between gap-4">
            <p className="text-xs uppercase tracking-[0.45em]" style={{ color: 'var(--muted)' }}>Hosted Login</p>
            <ThemeToggle />
          </div>
          <h1 className="mt-6 text-5xl font-semibold leading-tight" style={{ color: 'var(--foreground)' }}>Hosted authentication for your app, ready to plug in.</h1>
          <p className="mt-6 max-w-xl text-base" style={{ color: 'var(--muted)' }}>Use this page to handle sign in, registration, password reset, and email verification with AuthService. Connect your app ID, redirect URI, and provider credentials, then send users here when they need to authenticate.</p>
          <div className="mt-10 grid gap-4 md:grid-cols-3">
            {trustPoints.map(([Icon, title, copy]) => {
              const RenderIcon = Icon as typeof LockKeyhole;
              return (
                <div key={title as string} className="rounded-[1.2rem] border p-5" style={{ borderColor: 'var(--border)', background: 'var(--background-alt)' }}>
                  <RenderIcon className="h-5 w-5" style={{ color: 'var(--foreground)' }} />
                  <p className="mt-4 text-sm font-medium" style={{ color: 'var(--foreground)' }}>{title as string}</p>
                  <p className="mt-2 text-sm leading-6" style={{ color: 'var(--muted)' }}>{copy as string}</p>
                </div>
              );
            })}
          </div>
        </Panel>

        <Panel className="lg:min-h-[640px]">
          <p className="text-xs uppercase tracking-[0.35em]" style={{ color: 'var(--muted)' }}>
            {configState.status === 'ready' ? configState.config.name || appId : (appId || 'Missing App ID')}
          </p>
          <h2 className="mt-4 text-3xl font-semibold" style={{ color: 'var(--foreground)' }}>{titleMap[mode]}</h2>
          <p className="mt-3 text-sm" style={{ color: 'var(--muted)' }}>
            {redirectUri ? `After success, you will be redirected to ${redirectUri}.` : 'Provide app_id and redirect_uri query parameters to use this hosted login page.'}
          </p>
          <div className="mt-8 rounded-[1.2rem] border p-6" style={{ borderColor: 'var(--border)', background: 'var(--background-alt)' }}>
            {configState.status === 'loading' ? <p className="text-sm" style={{ color: 'var(--muted)' }}>Loading sign-in options...</p> : null}
            {configState.status === 'unavailable' ? <p className="text-sm" style={{ color: 'var(--muted)' }}>This application is unavailable.</p> : null}
            {configState.status === 'ready' && emailModesBlocked ? <EmailDisabledNotice appId={appId} redirectUri={redirectUri} /> : null}
            {configState.status === 'ready' && !emailModesBlocked && mode === 'login' && appId && redirectUri ? (
              <LoginForm appId={appId} redirectUri={redirectUri} authMethods={authMethods} />
            ) : null}
            {configState.status === 'ready' && !emailModesBlocked && mode === 'register' && appId && redirectUri ? (
              <RegisterForm appId={appId} redirectUri={redirectUri} />
            ) : null}
            {configState.status === 'ready' && !emailModesBlocked && mode === 'forgot' && appId ? (
              <ForgotPasswordForm appId={appId} redirectUri={redirectUri} />
            ) : null}
            {configState.status === 'ready' && !emailModesBlocked && mode === 'reset' && appId && token ? (
              <ResetPasswordForm appId={appId} token={token} redirectUri={redirectUri} />
            ) : null}
            {configState.status === 'ready' && mode === 'verify' && appId && token ? (
              <VerifyEmailView appId={appId} token={token} redirectUri={redirectUri} />
            ) : null}
            {missingParams && configState.status !== 'loading' && configState.status !== 'unavailable' ? (
              <p className="text-sm" style={{ color: 'var(--muted)' }}>Required query parameters are missing.</p>
            ) : null}
          </div>
        </Panel>
      </div>
    </main>
  );
}

export default function Page() {
  return (
    <Suspense fallback={<main className="flex min-h-screen items-center justify-center text-sm" style={{ color: 'var(--muted)' }}>Loading hosted login...</main>}>
      <HostedLoginInner />
    </Suspense>
  );
}
