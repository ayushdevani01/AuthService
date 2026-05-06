import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';

type AuthUser = {
  sub?: string;
  email?: string;
  aud?: string | string[];
  iss?: string;
  exp?: number;
  [key: string]: unknown;
};

type StoredSession = {
  accessToken: string;
  refreshToken?: string | null;
  expiresAt?: string | null;
  tokenType?: string | null;
};

type AuthContextValue = {
  user: AuthUser | null;
  isAuthenticated: boolean;
  loading: boolean;
  accessToken: string | null;
  login: (options?: { redirectUri?: string; mode?: 'login' | 'register' }) => void;
  logout: (options?: { redirectTo?: string }) => void;
  getAccessToken: () => string | null;
};

type AuthServiceProviderProps = {
  appId: string;
  authUrl: string;
  redirectUri: string;
  apiUrl?: string;
  storageKey?: string;
  children: React.ReactNode;
};

const AuthContext = createContext<AuthContextValue | null>(null);

function storageSessionKey(storageKey: string) {
  return `${storageKey}:session`;
}

function parseJwtPayload(token: string): AuthUser | null {
  try {
    const [, payload] = token.split('.');
    if (!payload) return null;
    const normalized = payload.replace(/-/g, '+').replace(/_/g, '/');
    const decoded = typeof window !== 'undefined' ? window.atob(normalized) : Buffer.from(normalized, 'base64').toString('utf8');
    return JSON.parse(decoded) as AuthUser;
  } catch {
    return null;
  }
}

function sessionExpired(session: StoredSession | null): boolean {
  if (!session?.accessToken) return true;
  if (session.expiresAt) {
    const expiresAt = Number(session.expiresAt);
    if (!Number.isNaN(expiresAt) && expiresAt * 1000 <= Date.now()) {
      return true;
    }
  }
  const payload = parseJwtPayload(session.accessToken);
  if (payload?.exp && payload.exp * 1000 <= Date.now()) {
    return true;
  }
  return false;
}

function readSession(storageKey: string): StoredSession | null {
  if (typeof window === 'undefined') return null;
  const raw = window.localStorage.getItem(storageSessionKey(storageKey));
  if (!raw) return null;
  try {
    return JSON.parse(raw) as StoredSession;
  } catch {
    return null;
  }
}

function writeSession(storageKey: string, session: StoredSession | null) {
  if (typeof window === 'undefined') return;
  const key = storageSessionKey(storageKey);
  if (!session) {
    window.localStorage.removeItem(key);
    return;
  }
  window.localStorage.setItem(key, JSON.stringify(session));
}

function resolveApiUrl(authUrl: string, apiUrl?: string) {
  if (apiUrl) return apiUrl.replace(/\/$/, '');
  if (typeof window !== 'undefined' && process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL.replace(/\/$/, '');
  }
  try {
    const parsed = new URL(authUrl);
    if (parsed.port === '3001') {
      parsed.port = '8080';
    }
    return parsed.origin;
  } catch {
    return 'http://localhost:8080';
  }
}

export function AuthServiceProvider({
  appId,
  authUrl,
  redirectUri,
  apiUrl,
  storageKey = 'authservice',
  children,
}: AuthServiceProviderProps) {
  const [session, setSession] = useState<StoredSession | null>(null);
  const [loading, setLoading] = useState(true);
  const resolvedApiUrl = useMemo(() => resolveApiUrl(authUrl, apiUrl), [authUrl, apiUrl]);

  useEffect(() => {
    const existing = readSession(storageKey);
    if (existing && sessionExpired(existing)) {
      writeSession(storageKey, null);
      setSession(null);
    } else if (existing) {
      setSession(existing);
    }
    setLoading(false);
  }, [storageKey]);

  const login = useCallback((options?: { redirectUri?: string; mode?: 'login' | 'register' }) => {
    const target = new URL(authUrl);
    target.searchParams.set('app_id', appId);
    target.searchParams.set('redirect_uri', options?.redirectUri || redirectUri);
    if (options?.mode) {
      target.searchParams.set('mode', options.mode);
    }
    window.location.href = target.toString();
  }, [appId, authUrl, redirectUri]);

  const logout = useCallback((options?: { redirectTo?: string }) => {
    const current = readSession(storageKey);
    if (current?.refreshToken) {
      void fetch(`${resolvedApiUrl}/oauth/revoke`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          token: current.refreshToken,
          token_type: 'refresh',
          app_id: appId,
        }),
      }).catch(() => undefined);
    }
    writeSession(storageKey, null);
    setSession(null);
    if (options?.redirectTo) {
      window.location.href = options.redirectTo;
    }
  }, [appId, resolvedApiUrl, storageKey]);

  const user = useMemo(() => {
    if (!session?.accessToken || sessionExpired(session)) return null;
    return parseJwtPayload(session.accessToken);
  }, [session]);

  const value = useMemo<AuthContextValue>(() => ({
    user,
    isAuthenticated: Boolean(session?.accessToken) && !sessionExpired(session),
    loading,
    accessToken: session?.accessToken || null,
    login,
    logout,
    getAccessToken: () => (session && !sessionExpired(session) ? session.accessToken : null),
  }), [user, session, loading, login, logout]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) {
    throw new Error('useAuth must be used inside AuthServiceProvider');
  }
  return value;
}

export function AuthGuard({ children, fallback = null }: { children: React.ReactNode; fallback?: React.ReactNode }) {
  const { isAuthenticated, loading } = useAuth();
  if (loading) return <>{fallback}</>;
  if (!isAuthenticated) return <>{fallback}</>;
  return <>{children}</>;
}

export function AuthCallbackHandler({ storageKey = 'authservice', onSuccess, fallback = null }: { storageKey?: string; onSuccess?: (session: StoredSession) => void; fallback?: React.ReactNode }) {
  useEffect(() => {
    if (typeof window === 'undefined') return;
    const hash = window.location.hash.startsWith('#') ? window.location.hash.slice(1) : window.location.hash;
    const params = new URLSearchParams(hash);
    const accessToken = params.get('access_token');
    if (!accessToken) return;

    const session: StoredSession = {
      accessToken,
      refreshToken: params.get('refresh_token'),
      tokenType: params.get('token_type'),
      expiresAt: params.get('expires_at'),
    };

    writeSession(storageKey, session);
    window.history.replaceState(null, '', `${window.location.pathname}${window.location.search}`);
    onSuccess?.(session);
  }, [storageKey, onSuccess]);

  return <>{fallback}</>;
}
