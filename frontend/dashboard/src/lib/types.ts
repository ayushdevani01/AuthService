export type Developer = {
  id: string;
  email: string;
  name: string;
  created_at: string;
  updated_at: string;
};

export type AppRecord = {
  id: string;
  app_id: string;
  public_app_id?: string;
  audience_app_id?: string;
  identifier_usage?: {
    public_app_id?: string;
    audience_app_id?: string;
  };
  developer_id: string;
  name: string;
  logo_url?: string;
  redirect_urls: string[];
  require_email_verification: boolean;
  created_at: string;
  updated_at: string;
};

export type AppIntegrationInfo = {
  public_app_id: string;
  audience_app_id: string;
  verify_api_key: string;
  jwks_url: string;
  verify_endpoint: string;
  userinfo_endpoint: string;
  notes: string[];
};

export type SigningKey = {
  id: string;
  app_id: string;
  kid: string;
  public_key: string;
  is_active: boolean;
  created_at: string;
  expires_at?: string | null;
  rotated_at?: string | null;
};

export type OAuthProvider = {
  id: string;
  app_id: string;
  provider: 'google' | 'github' | string;
  client_id: string;
  scopes: string[];
  enabled: boolean;
  created_at: string;
};

export type AppUser = {
  id: string;
  app_id: string;
  email: string;
  name?: string;
  avatar_url?: string;
  provider?: string;
  provider_user_id?: string;
  email_verified: boolean;
  created_at: string;
  last_login_at?: string;
};

export type PaginatedUsers = {
  users: AppUser[];
  next_page_token?: string;
  total_count: number;
};
