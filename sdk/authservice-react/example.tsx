import React from 'react';
import { AuthCallbackHandler, AuthGuard, AuthServiceProvider, useAuth } from './src';

function Home() {
  const { user, login, logout, isAuthenticated } = useAuth();

  return (
    <div>
      <h1>AuthService React SDK Example</h1>
      {isAuthenticated ? (
        <div>
          <p>Signed in as {user?.email || user?.sub}</p>
          <button onClick={() => logout()}>Logout</button>
        </div>
      ) : (
        <div>
          <button onClick={() => login()}>Login</button>
          <button onClick={() => login({ mode: 'register' })}>Register</button>
        </div>
      )}

      <AuthGuard fallback={<p>Please sign in to see protected content.</p>}>
        <p>Protected app content goes here.</p>
      </AuthGuard>
    </div>
  );
}

export default function App() {
  return (
    <AuthServiceProvider
      appId={process.env.REACT_APP_AUTH_APP_ID || 'app_your_public_app_id'}
      authUrl={process.env.REACT_APP_AUTH_URL || 'http://localhost:3001'}
      redirectUri={process.env.REACT_APP_AUTH_REDIRECT_URI || 'http://localhost:3000/auth/callback'}
    >
      <AuthCallbackHandler fallback={<p>Finishing sign-in...</p>} />
      <Home />
    </AuthServiceProvider>
  );
}
