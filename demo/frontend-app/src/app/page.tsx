'use client';

import { AuthServiceProvider, useAuth } from 'authservice-react';

function DemoHome() {
  const { isAuthenticated, loading, login, logout, user, refreshSession } = useAuth();

  if (loading) return <main><p>Loading session…</p></main>;

  return (
    <main>
      <h1>AuthService Demo</h1>
      <p>Uses one publishable key for hosted login and JWT audience.</p>
      {isAuthenticated ? (
        <div>
          <p>Signed in as {String(user?.email || user?.sub || 'user')}</p>
          <button type="button" onClick={() => void refreshSession()}>Refresh session</button>{' '}
          <button type="button" onClick={() => logout()}>Sign out</button>
        </div>
      ) : (
        <button type="button" onClick={() => login()}>Sign in</button>
      )}
    </main>
  );
}

export default function Page() {
  const appId = process.env.NEXT_PUBLIC_AUTH_PUBLISHABLE_KEY || '';
  const authUrl = process.env.NEXT_PUBLIC_AUTH_URL || 'http://localhost:3001';
  const redirectUri = process.env.NEXT_PUBLIC_AUTH_REDIRECT_URI || 'http://localhost:3000/callback';
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

  if (!appId) {
    return (
      <main>
        <h1>AuthService Demo</h1>
        <p>Set <code>NEXT_PUBLIC_AUTH_PUBLISHABLE_KEY</code> in <code>.env.local</code> from the dashboard.</p>
      </main>
    );
  }

  return (
    <AuthServiceProvider appId={appId} authUrl={authUrl} redirectUri={redirectUri} apiUrl={apiUrl}>
      <DemoHome />
    </AuthServiceProvider>
  );
}
