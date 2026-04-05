'use client';

import axios from 'axios';
import toast from 'react-hot-toast';
import { useAuthStore } from '@/store/auth';

const baseURL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export const api = axios.create({
  baseURL,
  headers: {
    'Content-Type': 'application/json',
  },
});

api.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (response) => response,
  (error) => {
    const status = error?.response?.status;
    if (status === 401) {
      useAuthStore.getState().logoutLocal();
      if (typeof window !== 'undefined' && window.location.pathname !== '/login') {
        toast.error('Session expired. Please sign in again.');
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  },
);
