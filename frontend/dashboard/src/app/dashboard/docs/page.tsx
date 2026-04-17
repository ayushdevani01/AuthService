import { ProtectedRoute } from '@/components/protected-route';
import { DashboardShell } from '@/components/shell';
import { BookOpenText, Braces, KeyRound, ShieldCheck } from 'lucide-react';
import { CodeBlock, CodeTabs } from '@/components/ui';

const sections = [
  {
    title: 'Integration contract',
    description: 'Every app currently has three values that matter during integration.',
    items: [
      'Public app ID: use for JWKS lookup, hosted login, and x-app-id on public verification requests.',
      'Audience app ID: use as the expected JWT aud value in your backend or Node SDK.',
      'API key: use for POST /api/v1/verify requests from your backend.',
    ],
  },
  {
    title: 'Hosted login flow',
    description: 'The hosted login UI expects the public app ID and a redirect URI.',
    items: [
      'Local login UI URL: http://localhost:3001',
      'Required query params: app_id and redirect_uri',
      'Use the same redirect URI in your AuthService app and the OAuth provider console',
    ],
  },
  {
    title: 'Backend verification',
    description: 'You can verify tokens with the Node SDK or through the verification API.',
    items: [
      'SDK verification uses the public app ID for JWKS and the audience app ID for aud validation.',
      'REST verification uses x-api-key plus x-app-id headers.',
      'Both verification paths validate signature, issuer, and audience.',
    ],
  },
  {
    title: 'React frontend SDK',
    description: 'The frontend SDK is meant to reduce integration to a provider, a callback handler, and a few hook calls.',
    items: [
      'Wrap your app with AuthServiceProvider using the public app ID, auth URL, and redirect URI.',
      'Render AuthCallbackHandler on your callback route to store returned tokens automatically.',
      'Use useAuth() for login, logout, and current user state.',
    ],
  },
  {
    title: 'Google OAuth setup',
    description: 'Use Google Cloud Console and make the callback URL match exactly between Google and AuthService.',
    items: [
      'Open Google Cloud Console, choose your project, then go to APIs & Services > Credentials.',
      'Click Create Credentials > OAuth client ID.',
      'If prompted, configure the OAuth consent screen first and save it.',
      'Choose Web application as the application type.',
      'Under Authorized redirect URIs, paste the same callback URL you added in AuthService.',
      'Copy the generated Client ID and Client Secret, then paste them into Dashboard > App > Providers > Google.',
    ],
  },
  {
    title: 'GitHub OAuth setup',
    description: 'Create an OAuth app in GitHub and use the exact same callback URL you registered in AuthService.',
    items: [
      'Open GitHub Settings > Developer settings > OAuth Apps.',
      'Click New OAuth App.',
      'Fill in Application name, Homepage URL, and Authorization callback URL.',
      'Paste the same callback URL you configured in AuthService into Authorization callback URL.',
      'Create the app, then copy the Client ID and generate a Client Secret.',
      'Paste both values into Dashboard > App > Providers > GitHub.',
    ],
  },
];

const envSnippet = `AUTH_APP_ID=app_your_public_app_id\nAUTH_AUDIENCE=your-internal-app-uuid\nAUTH_API_URL=http://localhost:8080\nAUTH_ISSUER=https://auth.yourplatform.com`;

const reactSnippet = `import { AuthCallbackHandler, AuthGuard, AuthServiceProvider, useAuth } from 'authservice-react';\n\nfunction LoginButton() {\n  const { login } = useAuth();\n  return <button onClick={() => login()}>Sign in</button>;\n}\n\nexport default function App() {\n  return (\n    <AuthServiceProvider\n      appId={process.env.NEXT_PUBLIC_AUTH_APP_ID!}\n      authUrl={process.env.NEXT_PUBLIC_AUTH_URL!}\n      redirectUri={process.env.NEXT_PUBLIC_AUTH_REDIRECT_URI!}\n    >\n      <AuthCallbackHandler />\n      <AuthGuard fallback={<LoginButton />}>\n        <div>Protected app content</div>\n      </AuthGuard>\n    </AuthServiceProvider>\n  );\n}`;

const nodeSnippet = `import { requireAuth } from 'authservice-node';\n\napp.get('/protected', requireAuth({\n  appId: process.env.AUTH_APP_ID,\n  audience: process.env.AUTH_AUDIENCE,\n  apiUrl: process.env.AUTH_API_URL,\n  issuer: process.env.AUTH_ISSUER,\n}), (req, res) => {\n  res.json({ user: req.auth });\n});`;

const verifySnippet = `curl -X POST http://localhost:8080/api/v1/verify \\\n+  -H "Content-Type: application/json" \\\n+  -H "x-api-key: <your-api-key>" \\\n+  -H "x-app-id: <your-public-app-id>" \\\n+  -d '{\n    "token": "<jwt>",\n    "app_id": "<your-public-app-id>"\n  }'`;

export default function DashboardDocsPage() {
  return (
    <ProtectedRoute>
      <DashboardShell>
        <div className="space-y-6">
          <section className="glass-panel rounded-[1.9rem] p-8 lg:p-10">
            <div className="max-w-3xl space-y-4">
              <div className="inline-flex items-center gap-2 rounded-full border border-[var(--border)] bg-[var(--background-alt)] px-3 py-1.5 text-xs text-muted">
                <BookOpenText className="h-3.5 w-3.5" />
                Dashboard documentation
              </div>
              <h1 className="text-4xl font-semibold tracking-tight text-foreground">Integration docs that match the product</h1>
              <p className="text-sm leading-7 text-muted lg:text-base">
                This page mirrors the current backend contract exactly so developers can wire up hosted login, token verification, and app identifiers without guessing which value belongs where.
              </p>
            </div>
          </section>

          <section className="grid gap-5 xl:grid-cols-[260px_minmax(0,1fr)]">
            <aside className="luxury-card hidden h-fit rounded-[1.75rem] p-5 xl:block">
              <p className="text-xs uppercase tracking-[0.35em] text-muted">On this page</p>
              <nav className="mt-5 space-y-2 text-sm">
                {sections.map((section) => (
                  <a key={section.title} href={`#${section.title.toLowerCase().replace(/\s+/g, '-')}`} className="block rounded-xl px-3 py-2 text-muted transition hover:bg-[var(--background-alt)] hover:text-foreground">
                    {section.title}
                  </a>
                ))}
              </nav>
            </aside>

            <div className="space-y-5">
              {sections.map((section, index) => (
                <article id={section.title.toLowerCase().replace(/\s+/g, '-')} key={section.title} className="luxury-card rounded-[1.75rem] p-7 lg:p-8">
                  <div className="flex items-center gap-3 text-xs uppercase tracking-[0.35em] text-muted">
                    <span>{String(index + 1).padStart(2, '0')}</span>
                    <span>{section.title}</span>
                  </div>
                  <p className="mt-4 max-w-3xl text-sm leading-7 text-muted">{section.description}</p>
                  <div className="mt-6 space-y-3">
                    {section.items.map((item) => (
                      <div key={item} className="rounded-2xl border border-[var(--border)] bg-[var(--background-alt)] px-4 py-4 text-sm leading-6 text-foreground">
                        {item}
                      </div>
                    ))}
                  </div>
                </article>
              ))}
            </div>
          </section>

          <section className="luxury-card rounded-[1.75rem] p-6 lg:p-7">
            <div className="flex items-center gap-3 text-foreground">
              <Braces className="h-5 w-5" />
              <h2 className="text-lg font-medium">Reference code</h2>
            </div>
            <div className="mt-5">
              <CodeTabs
                defaultTab="env"
                tabs={[
                  { key: 'env', label: 'Environment', code: envSnippet },
                  { key: 'react', label: 'React SDK', code: reactSnippet },
                  { key: 'node', label: 'Node SDK', code: nodeSnippet },
                  { key: 'verify', label: 'Verify API', code: verifySnippet },
                ]}
              />
            </div>
          </section>

          <section className="luxury-card rounded-[1.75rem] p-7 lg:p-8">
            <div className="max-w-3xl">
              <p className="text-xs uppercase tracking-[0.35em] text-muted">Frontend SDK</p>
              <h2 className="mt-3 text-3xl font-semibold text-foreground">React integration in a few lines</h2>
              <p className="mt-3 text-sm leading-7 text-muted">
                Use the React SDK when you want hosted login plus session handling without manually building redirect URLs, parsing callback fragments, or storing tokens yourself.
              </p>
            </div>
            <div className="mt-6">
              <CodeBlock title="React SDK" code={reactSnippet} />
            </div>
          </section>
        </div>
      </DashboardShell>
    </ProtectedRoute>
  );
}
