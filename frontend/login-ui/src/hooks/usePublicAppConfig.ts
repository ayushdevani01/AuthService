'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import type { PublicAppConfig } from '@/lib/types';

type State =
  | { status: 'idle' | 'loading' }
  | { status: 'ready'; config: PublicAppConfig }
  | { status: 'unavailable' };

export function usePublicAppConfig(appId: string) {
  const [state, setState] = useState<State>({ status: appId ? 'loading' : 'idle' });

  useEffect(() => {
    if (!appId) {
      setState({ status: 'idle' });
      return;
    }

    let cancelled = false;
    setState({ status: 'loading' });

    api
      .get(`/api/v1/public/apps/${encodeURIComponent(appId)}`)
      .then(({ data }) => {
        if (cancelled) return;
        setState({ status: 'ready', config: data as PublicAppConfig });
      })
      .catch(() => {
        if (cancelled) return;
        setState({ status: 'unavailable' });
      });

    return () => {
      cancelled = true;
    };
  }, [appId]);

  return state;
}
