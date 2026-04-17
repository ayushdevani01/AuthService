import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';

type AuthUser = {
  sub?: string;
  email?: string;
  aud?: string | string[];
  iss?: string;
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

export function AuthServiceProvider({ appId, authUrl, redirectUri, storageKey = 'authservice', children }: AuthServiceProviderProps) {
  const [session, setSession] = useState<StoredSession | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const existing = readSession(storageKey);
    if (existing) {
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
    writeSession(storageKey, null);
    setSession(null);
    if (options?.redirectTo) {
      window.location.href = options.redirectTo;
    }
  }, [storageKey]);

  const user = useMemo(() => {
    if (!session?.accessToken) return null;
    return parseJwtPayload(session.accessToken);
  }, [session?.accessToken]);

  const value = useMemo<AuthContextValue>(() => ({
    user,
    isAuthenticated: Boolean(session?.accessToken),
    loading,
    accessToken: session?.accessToken || null,
    login,
    logout,
    getAccessToken: () => session?.accessToken || null,
  }), [user, session?.accessToken, loading, login, logout]);

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
    onSuccess?.(session);
  }, [storageKey, onSuccess]);

  return <>{fallback}</>;
}
