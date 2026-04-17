import Link from 'next/link';
import { ArrowRight, BookOpenText, Braces, KeyRound, ShieldCheck, Workflow } from 'lucide-react';
import { CodeTabs, ThemeToggle } from '@/components/ui';

const valueProps = [
  {
    title: 'Everything for auth in one workspace',
    description: 'Create apps, configure providers, rotate keys, inspect users, and integrate your backend without hopping between tools.',
    icon: Workflow,
  },
  {
    title: 'A quickstart that answers real questions',
    description: 'The docs focus on what developers actually need to wire up: which ID goes where, which secrets are one-time, and how verification works.',
    icon: BookOpenText,
  },
  {
    title: 'Verification designed for production',
    description: 'Use RS256 tokens, JWKS lookup, audience validation, and API-key protected backend verification endpoints.',
    icon: ShieldCheck,
  },
];

const setupSteps = [
  {
    step: '01',
    title: 'Create an app in the dashboard',
    description: 'You will receive a public app ID, an internal audience UUID, and a one-time API key.',
  },
  {
    step: '02',
    title: 'Register redirect URLs and providers',
    description: 'Add every callback URL you use locally and in production, then connect Google or GitHub credentials.',
  },
  {
    step: '03',
    title: 'Add the right values to your app',
    description: 'Use the public app ID for JWKS and hosted login. Use the internal UUID as the JWT audience.',
  },
  {
    step: '04',
    title: 'Verify tokens in your backend',
    description: 'Use the Node SDK or call the verification API with your API key and public app ID.',
  },
];

const codeTabs = {
  env: `AUTH_APP_ID=app_your_public_app_id\nAUTH_AUDIENCE=your-internal-app-uuid\nAUTH_API_URL=http://localhost:8080\nAUTH_ISSUER=https://auth.yourplatform.com`,
  react: `import { AuthCallbackHandler, AuthGuard, AuthServiceProvider, useAuth } from 'authservice-react';\n\nfunction LoginButton() {\n  const { login } = useAuth();\n  return <button onClick={() => login()}>Sign in</button>;\n}\n\nexport default function App() {\n  return (\n    <AuthServiceProvider\n      appId={process.env.NEXT_PUBLIC_AUTH_APP_ID!}\n      authUrl={process.env.NEXT_PUBLIC_AUTH_URL!}\n      redirectUri={process.env.NEXT_PUBLIC_AUTH_REDIRECT_URI!}\n    >\n      <AuthCallbackHandler />\n      <AuthGuard fallback={<LoginButton />}>\n        <div>Protected app content</div>\n      </AuthGuard>\n    </AuthServiceProvider>\n  );\n}`,
  node: `import { requireAuth } from 'authservice-node';\n\napp.get('/protected', requireAuth({\n  appId: process.env.AUTH_APP_ID,\n  audience: process.env.AUTH_AUDIENCE,\n  apiUrl: process.env.AUTH_API_URL,\n  issuer: process.env.AUTH_ISSUER,\n}), (req, res) => {\n  res.json({ user: req.auth });\n});`,
  curl: `curl -X POST http://localhost:8080/api/v1/verify \\\n+  -H "Content-Type: application/json" \\\n+  -H "x-api-key: <your-api-key>" \\\n+  -H "x-app-id: <your-public-app-id>" \\\n+  -d '{\n    "token": "<jwt>",\n    "app_id": "<your-public-app-id>"\n  }'`,
};

const docSections = [
  {
    id: 'identifiers',
    title: 'Know your identifiers',
    body: 'This platform intentionally uses two app identifiers. The public app ID is for JWKS lookup and public API requests. The internal UUID is the JWT audience value.',
    bullets: [
      '`AUTH_APP_ID` = public app ID like `app_xxx`',
      '`AUTH_AUDIENCE` = internal app UUID used in `aud`',
      '`x-api-key` = backend verification API credential',
    ],
  },
  {
    id: 'oauth',
    title: 'Configure OAuth correctly',
    body: 'Your callback URL must match across AuthService and the provider console. Treat local and production environments as separate entries.',
    bullets: [
      'In Google Cloud Console, go to APIs & Services > Credentials > Create Credentials > OAuth client ID.',
      'Choose Web application, then paste your AuthService callback URL into Authorized redirect URIs.',
      'In GitHub, go to Settings > Developer settings > OAuth Apps > New OAuth App.',
      'Paste the same callback URL into Authorization callback URL, then copy the generated Client ID and Client Secret into the dashboard provider form.',
    ],
  },
  {
    id: 'hosted-login',
    title: 'Launch hosted login fast',
    body: 'The hosted login UI receives the public app ID plus the redirect URI, then returns tokens back to your application callback route.',
    bullets: [
      'Hosted login runs on `http://localhost:3001` in local development',
      'Pass `app_id` and `redirect_uri` query params',
      'Read returned tokens in your callback page and establish your session',
      'Use the React SDK if you want the provider, callback handler, and auth state management done for you',
    ],
  },
  {
    id: 'react-sdk',
    title: 'Use the React SDK for the frontend',
    body: 'If your app is React or Next.js client-side, the frontend SDK removes most manual auth wiring. You only provide the public app ID, auth URL, and redirect URI.',
    bullets: [
      'Wrap your app in `AuthServiceProvider`.',
      'Render `AuthCallbackHandler` on the callback route.',
      'Call `useAuth()` for login, logout, access token, and user state.',
      'Protect app content with `AuthGuard`.',
    ],
  },
  {
    id: 'keys',
    title: 'Handle keys like production secrets',
    body: 'API keys are only shown once, and signing keys should be rotated with a grace period to avoid breaking live sessions.',
    bullets: [
      'Save the API key immediately after app creation',
      'Rotate API keys if they are ever exposed',
      'Rotate signing keys with grace periods during rollouts',
    ],
  },
];

export default function HomePage() {
  return (
    <main className="min-h-screen px-4 py-6 lg:px-6">
      <div className="mx-auto max-w-[1380px] space-y-6">
        <header className="glass-panel rounded-[1.75rem] px-6 py-5">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div className="flex items-center gap-4">
              <div className="flex h-11 w-11 items-center justify-center rounded-2xl border border-[var(--border)] bg-app-panel text-sm font-semibold text-foreground">AS</div>
              <div>
                <p className="text-xs uppercase tracking-[0.35em] text-muted">AuthService</p>
                <p className="text-sm text-muted">Authentication infrastructure with a dashboard, hosted login, and backend verification.</p>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <ThemeToggle />
              <Link href="#quickstart" className="button-secondary">Quickstart</Link>
              <Link href="/login" className="button-secondary">Developer Login</Link>
              <Link href="/dashboard" className="button-primary">Open Dashboard</Link>
            </div>
          </div>
        </header>

        <section className="grid gap-5 xl:grid-cols-[1.15fr_0.85fr]">
          <div className="glass-panel rounded-[1.9rem] px-7 py-8 lg:px-10 lg:py-10">
            <div className="max-w-3xl space-y-6">
              <div className="inline-flex items-center gap-2 rounded-full border border-[var(--border)] bg-[var(--background-alt)] px-3 py-1.5 text-xs text-muted">
                <BookOpenText className="h-3.5 w-3.5" />
                Docs-first auth platform
              </div>
              <div className="space-y-4">
                <h1 className="max-w-4xl text-4xl font-semibold tracking-tight text-foreground lg:text-6xl">
                  Docs that explain the integration, not just the product.
                </h1>
                <p className="max-w-2xl text-base leading-7 text-muted lg:text-lg">
                  AuthService gives product teams a cleaner path from app creation to production-ready token verification. The homepage now mirrors the way strong docs products work: fast context, clear setup order, and real implementation examples.
                </p>
              </div>
              <div className="flex flex-wrap gap-3">
                <Link href="#quickstart" className="button-primary">
                  Start with the quickstart
                  <ArrowRight className="ml-2 h-4 w-4" />
                </Link>
                <Link href="/dashboard" className="button-secondary">See the developer dashboard</Link>
              </div>
            </div>
          </div>

          <aside className="luxury-card rounded-[1.9rem] p-6 lg:p-7">
            <div className="space-y-4">
              <p className="text-xs uppercase tracking-[0.35em] text-muted">Integration Snapshot</p>
              <h2 className="text-2xl font-semibold text-foreground">What developers need on day one</h2>
              <div className="space-y-3">
                <div className="rounded-2xl border border-[var(--border)] bg-[var(--background-alt)] p-4">
                  <p className="text-xs uppercase tracking-[0.3em] text-muted">Public App ID</p>
                  <p className="mt-2 text-sm text-foreground">Used for JWKS lookup, hosted login, and public API requests.</p>
                </div>
                <div className="rounded-2xl border border-[var(--border)] bg-[var(--background-alt)] p-4">
                  <p className="text-xs uppercase tracking-[0.3em] text-muted">Audience UUID</p>
                  <p className="mt-2 text-sm text-foreground">Used as the expected JWT `aud` value in backend verification.</p>
                </div>
                <div className="rounded-2xl border border-[var(--border)] bg-[var(--background-alt)] p-4">
                  <p className="text-xs uppercase tracking-[0.3em] text-muted">Verify API Key</p>
                  <p className="mt-2 text-sm text-foreground">Required when calling the REST verification endpoint from your backend.</p>
                </div>
              </div>
            </div>
          </aside>
        </section>

        <section className="grid gap-4 md:grid-cols-3">
          {valueProps.map((item) => {
            const Icon = item.icon;
            return (
              <div key={item.title} className="luxury-card rounded-[1.5rem] p-6">
                <div className="flex h-11 w-11 items-center justify-center rounded-2xl border border-[var(--border)] bg-[var(--background-alt)] text-foreground">
                  <Icon className="h-5 w-5" />
                </div>
                <h3 className="mt-5 text-lg font-medium text-foreground">{item.title}</h3>
                <p className="mt-2 text-sm leading-6 text-muted">{item.description}</p>
              </div>
            );
          })}
        </section>

        <section id="quickstart" className="grid gap-5 xl:grid-cols-[0.82fr_1.18fr]">
          <div className="luxury-card rounded-[1.9rem] p-7">
            <p className="text-xs uppercase tracking-[0.35em] text-muted">Quickstart</p>
            <h2 className="mt-3 text-3xl font-semibold text-foreground">A setup flow developers can scan in one minute</h2>
            <div className="mt-8 space-y-5">
              {setupSteps.map((item) => (
                <div key={item.step} className="flex gap-4">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl border border-[var(--border)] bg-[var(--background-alt)] text-sm font-semibold text-foreground">
                    {item.step}
                  </div>
                  <div>
                    <p className="text-base font-medium text-foreground">{item.title}</p>
                    <p className="mt-1 text-sm leading-6 text-muted">{item.description}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="luxury-card rounded-[1.9rem] p-7">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div>
                <p className="text-xs uppercase tracking-[0.35em] text-muted">Reference Snippets</p>
                <h2 className="mt-3 text-3xl font-semibold text-foreground">Copy the contract exactly</h2>
              </div>
              <div className="inline-flex items-center gap-2 rounded-full border border-[var(--border)] bg-[var(--background-alt)] px-3 py-1.5 text-xs text-muted">
                <Braces className="h-3.5 w-3.5" />
                Node SDK + REST verify
              </div>
            </div>
            <div className="mt-6">
              <CodeTabs
                defaultTab="env"
                tabs={[
                  { key: 'env', label: 'Environment', code: codeTabs.env },
                  { key: 'react', label: 'React SDK', code: codeTabs.react },
                  { key: 'node', label: 'Node SDK', code: codeTabs.node },
                  { key: 'curl', label: 'Verify API', code: codeTabs.curl },
                ]}
              />
            </div>
          </div>
        </section>

        <section id="docs" className="grid gap-5 xl:grid-cols-[260px_minmax(0,1fr)]">
          <aside className="luxury-card hidden h-fit rounded-[1.75rem] p-5 xl:block xl:sticky xl:top-6">
            <p className="text-xs uppercase tracking-[0.35em] text-muted">In this guide</p>
            <nav className="mt-5 space-y-2 text-sm">
              {docSections.map((section) => (
                <a key={section.id} href={`#${section.id}`} className="block rounded-xl px-3 py-2 text-muted transition hover:bg-[var(--background-alt)] hover:text-foreground">
                  {section.title}
                </a>
              ))}
            </nav>
          </aside>

          <div className="space-y-5">
            {docSections.map((section) => (
              <article id={section.id} key={section.id} className="luxury-card rounded-[1.75rem] p-7 lg:p-8">
                <h3 className="text-2xl font-semibold text-foreground">{section.title}</h3>
                <p className="mt-3 max-w-3xl text-sm leading-7 text-muted">{section.body}</p>
                <div className="mt-6 grid gap-3 md:grid-cols-1">
                  {section.bullets.map((bullet) => (
                    <div key={bullet} className="rounded-2xl border border-[var(--border)] bg-[var(--background-alt)] px-4 py-4 text-sm leading-6 text-foreground">
                      {bullet}
                    </div>
                  ))}
                </div>
              </article>
            ))}
          </div>
        </section>

        <section className="glass-panel rounded-[1.75rem] px-7 py-8 lg:px-10">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.35em] text-muted">Next Step</p>
              <h2 className="mt-2 text-3xl font-semibold text-foreground">Create an app and copy the integration values from the dashboard</h2>
              <p className="mt-3 max-w-2xl text-sm leading-7 text-muted">
                The dashboard now surfaces the public app ID, audience UUID, and API key separately so developers do not have to guess which value belongs in the SDK, JWKS route, or verify API.
              </p>
            </div>
            <div className="flex flex-wrap gap-3">
              <Link href="/register" className="button-secondary">Create developer account</Link>
              <Link href="/dashboard" className="button-primary">
                Open dashboard
                <ArrowRight className="ml-2 h-4 w-4" />
              </Link>
            </div>
          </div>
        </section>
      </div>
    </main>
  );
}
