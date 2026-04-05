'use client';

import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { Developer } from '@/lib/types';

type AuthState = {
  accessToken: string | null;
  developer: Developer | null;
  hydrated: boolean;
  setAuth: (payload: { accessToken: string; developer: Developer }) => void;
  updateDeveloper: (developer: Developer) => void;
  logoutLocal: () => void;
  markHydrated: () => void;
};

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      accessToken: null,
      developer: null,
      hydrated: false,
      setAuth: ({ accessToken, developer }) => set({ accessToken, developer }),
      updateDeveloper: (developer) => set({ developer }),
      logoutLocal: () => set({ accessToken: null, developer: null }),
      markHydrated: () => set({ hydrated: true }),
    }),
    {
      name: 'authservice-dashboard-auth',
      storage: createJSONStorage(() => localStorage),
      skipHydration: true,
      partialize: (state) => ({
        accessToken: state.accessToken,
        developer: state.developer,
      }),
      onRehydrateStorage: () => (state) => {
        state?.markHydrated();
      },
    },
  ),
);
