import Link from 'next/link';
import { ArrowRight, BookOpenText, KeyRound, ShieldCheck, Workflow } from 'lucide-react';
import { ThemeToggle } from '@/components/ui';

const features = [
  {
    title: 'Ship authentication surfaces faster',
    description: 'Create apps, manage OAuth providers, and provision hosted login flows with an interface tuned for developer teams.',
    icon: KeyRound,
  },
  {
    title: 'Control every identity edge',
    description: 'Inspect redirect URLs, email verification rules, API keys, signing keys, and end-user activity from one workspace.',
    icon: ShieldCheck,
  },
  {
    title: 'Integrate with clear docs',
    description: 'Move from app creation to implementation using a docs section designed around real setup steps instead of marketing fluff.',
    icon: BookOpenText,
  },
];

const docs = [
  {
    title: '1. Create your app first',
    points: [
      'In the dashboard, create an application and copy its `app_id`.',
      'Add every callback URL you will use, including local development and production URLs.',
      'If your app uses hosted login locally, a common redirect URI is `http://localhost:3000/auth/callback`.',
      'Turn on “Require verified email” if users must verify before email/password login succeeds.',
    ],
  },
  {
    title: '2. What to put in redirect URI',
    points: [
      'Use the exact frontend route where your application receives auth results after login.',
      'For local dashboard testing, `http://localhost:3000/auth/callback` is the expected callback route in this project.',
      'Add the same URI both in AuthService app settings and in the Google or GitHub OAuth provider console.',
      'If you use multiple environments, add each environment-specific redirect URI separately.',
    ],
  },
  {
    title: '3. Google OAuth credentials',
    points: [
      'In Google Cloud Console, create an OAuth 2.0 Client ID for a Web application.',
      'Set the authorized redirect URI to the same callback you added in AuthService, for example `http://localhost:3000/auth/callback`.',
      'Copy the Google Client ID and Client Secret into the Providers tab for your AuthService app.',
      'Use the standard Google scopes shown in the dashboard: `openid`, `email`, and `profile`.',
    ],
  },
  {
    title: '4. GitHub OAuth credentials',
    points: [
      'In GitHub Developer Settings, create a new OAuth App.',
      'Set the Authorization callback URL to the exact same redirect URI registered in your AuthService app.',
      'Copy the GitHub Client ID and Client Secret into the Providers tab for your app.',
      'Use the dashboard scope presets such as `read:user` and `user:email` unless your product needs more.',
    ],
  },
  {
    title: '5. Hosted login integration',
    points: [
      'Open the hosted login UI on port `3001` with `app_id` and `redirect_uri` query params.',
      'Example: `http://localhost:3001/?app_id=YOUR_APP_ID&redirect_uri=http://localhost:3000/auth/callback`.',
      'After success, AuthService redirects back with tokens in the URL hash.',
      'Your frontend callback page should read the hash and establish the session for your app.',
    ],
  },
  {
    title: '6. Keys and production setup',
    points: [
      'Store the first API key shown at app creation immediately because it is only revealed once.',
      'Rotate API keys whenever a secret may have been exposed.',
      'Rotate signing keys with a grace period so older tokens keep working temporarily during rollout.',
      'Use the “Include Expired” toggle to inspect both active and expired signing keys when debugging.',
    ],
  },
];

export default function HomePage() {
  return (
    <main className="min-h-screen px-4 py-6 lg:px-6">
      <div className="mx-auto max-w-[1380px] space-y-5">
        <header className="glass-panel rounded-[1.6rem] px-6 py-5">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div className="flex items-center gap-4">
              <div className="flex h-11 w-11 items-center justify-center rounded-xl border border-[var(--border)] bg-app-panel text-foreground">A</div>
              <div>
                <p className="text-xs uppercase tracking-[0.35em] text-muted">AuthService</p>
                <p className="text-sm text-muted">Premium authentication infrastructure for product teams</p>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <ThemeToggle />
              <Link href="#docs" className="button-secondary">Docs</Link>
              <Link href="/login" className="button-secondary">Developer Login</Link>
              <Link href="/dashboard" className="button-primary">Open Dashboard</Link>
            </div>
          </div>
        </header>

        <section className="glass-panel overflow-hidden rounded-[1.6rem] px-6 py-12 lg:px-10 lg:py-16">
          <div className="grid gap-10 lg:grid-cols-[1.1fr_0.9fr] lg:items-end">
            <div>
              <p className="text-xs uppercase tracking-[0.45em] text-muted">Developer-First Identity</p>
              <h1 className="mt-6 max-w-4xl text-5xl font-semibold leading-[0.96] text-foreground lg:text-7xl">Authentication infrastructure with the calm precision of a premium developer tool.</h1>
              <p className="mt-7 max-w-2xl text-[15px] leading-7 text-muted lg:text-[17px]">Build original authentication workflows with a dark-first marketing surface, light-mode dashboard support, clear integration steps, and focused operational controls.</p>
              <div className="mt-8 flex flex-wrap items-center gap-3">
                <Link href="/dashboard" className="button-primary"><ArrowRight className="mr-2 h-4 w-4" />Start in dashboard</Link>
                <Link href="#docs" className="button-secondary">Read the docs</Link>
              </div>
            </div>

            <div className="rounded-[1.25rem] border border-[var(--border)] bg-app-panel p-5">
              <div className="rounded-[1.1rem] border border-[var(--border)] bg-[var(--background-alt)] p-5">
                <div className="flex items-center justify-between border-b border-[var(--border)] pb-4">
                  <div>
                    <p className="text-sm font-medium text-foreground">Identity Control Plane</p>
                    <p className="text-sm text-muted">Apps, OAuth, users, keys, and hosted login</p>
                  </div>
                  <Workflow className="h-5 w-5 text-muted" />
                </div>
                <div className="mt-5 space-y-3 text-sm">
                  {[
                    'Create an app and store the first API key securely',
                    'Enable Google or GitHub with curated scopes',
                    'Choose email verification rules per app',
                    'Use hosted login with redirect hashes for tokens',
                  ].map((item, index) => (
                    <div key={item} className="flex items-start gap-3 rounded-xl border border-[var(--border)] px-4 py-3 text-muted">
                      <span className="w-5 text-[11px] uppercase tracking-[0.22em] text-foreground/80">0{index + 1}</span>
                      <span>{item}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </section>

        <section className="grid gap-4 lg:grid-cols-3">
          {features.map(({ title, description, icon: Icon }) => (
            <article key={title} className="luxury-card p-6">
              <Icon className="h-5 w-5 text-foreground" />
              <h2 className="mt-5 text-[1.15rem] font-medium text-foreground">{title}</h2>
              <p className="mt-3 text-sm leading-6 text-muted">{description}</p>
            </article>
          ))}
        </section>

        <section id="docs" className="glass-panel rounded-[1.6rem] px-6 py-10 lg:px-10">
          <div className="max-w-3xl">
            <p className="text-xs uppercase tracking-[0.45em] text-muted">Docs</p>
            <h2 className="mt-4 max-w-4xl text-4xl font-semibold leading-tight text-foreground">Everything a developer needs to configure OAuth correctly</h2>
            <p className="mt-4 text-base leading-7 text-muted">This docs section is written to answer the practical setup questions developers actually hit: what redirect URI to use, where to create credentials, what values to paste into providers, and how hosted login behaves.</p>
          </div>
          <div className="mt-8 grid gap-4 lg:grid-cols-2 xl:grid-cols-3">
            {docs.map((section) => (
              <article key={section.title} className="rounded-[1.1rem] border border-[var(--border)] bg-app-panel p-6">
                <h3 className="text-lg font-medium text-foreground">{section.title}</h3>
                <ul className="mt-4 space-y-3 text-sm leading-6 text-muted">
                  {section.points.map((point) => (
                    <li key={point} className="rounded-xl border border-[var(--border)] px-4 py-3">{point}</li>
                  ))}
                </ul>
              </article>
            ))}
          </div>
        </section>

        <footer className="glass-panel rounded-[1.6rem] px-6 py-8 lg:px-10">
          <div className="flex flex-col gap-6 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.35em] text-muted">AuthService</p>
              <p className="mt-3 max-w-2xl text-sm leading-6 text-muted">A developer-focused authentication platform with original functionality and a premium monochrome presentation across marketing, docs, dashboard, and hosted login.</p>
            </div>
            <div className="flex flex-wrap gap-3">
              <Link href="/dashboard" className="button-primary">Dashboard</Link>
              <Link href="/login" className="button-secondary">Developer Sign In</Link>
            </div>
          </div>
        </footer>
      </div>
    </main>
  );
}
