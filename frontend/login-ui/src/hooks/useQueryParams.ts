'use client';

import { useMemo } from 'react';
import { useSearchParams } from 'next/navigation';
import type { QueryMode } from '@/lib/types';

export function useQueryParams() {
  const searchParams = useSearchParams();

  return useMemo(() => {
    const mode = (searchParams.get('mode') || 'login') as QueryMode;
    return {
      appId: searchParams.get('app_id') || '',
      redirectUri: searchParams.get('redirect_uri') || '',
      token: searchParams.get('token') || '',
      codeChallenge: searchParams.get('code_challenge') || '',
      codeChallengeMethod: searchParams.get('code_challenge_method') || 'S256',
      mode,
    };
  }, [searchParams]);
}
