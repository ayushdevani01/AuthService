export type AuthTokens = {
  access_token: string;
  refresh_token?: string;
  token_type: string;
  expires_at: number;
  refresh_token_expires_at?: number | null;
  verification_sent?: boolean;
  requires_verification?: boolean;
  message?: string;
};

export type QueryMode = 'login' | 'register' | 'forgot' | 'reset' | 'verify';
