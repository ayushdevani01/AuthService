'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/store/auth';
import type { ReactNode } from 'react';

export function ProtectedRoute({ children }: { children: ReactNode }) {
  const router = useRouter();
  const accessToken = useAuthStore((state) => state.accessToken);
  const hydrated = useAuthStore((state) => state.hydrated);

  useEffect(() => {
    if (hydrated && !accessToken) {
      router.replace('/login');
    }
  }, [accessToken, hydrated, router]);

  if (!hydrated) {
    return <div className="flex min-h-screen items-center justify-center text-sm text-muted">Loading session...</div>;
  }

  if (!accessToken) {
    return null;
  }

  return <>{children}</>;
}
