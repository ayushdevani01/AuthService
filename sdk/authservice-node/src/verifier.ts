import * as jose from 'jose';

export interface AuthOptions {
  appId?: string;
  /** @deprecated Prefer appId. Still accepted for one-release dual-accept of legacy tokens. */
  audience?: string;
  apiUrl?: string;
  issuer?: string;
}

export interface AuthPayload {
  userId: string;
  appId: string;
  email?: string;
  [key: string]: any;
}

export class AuthVerifier {
  private appId: string;
  private audiences: string[];
  private apiUrl: string;
  private issuer: string;
  private jwksUrl: URL;
  private jwks: ReturnType<typeof jose.createRemoteJWKSet>;

  constructor(options?: AuthOptions) {
    this.appId = options?.appId || process.env.AUTH_APP_ID || process.env.AUTH_PUBLISHABLE_KEY || '';
    const legacyAudience = options?.audience || process.env.AUTH_AUDIENCE || '';
    this.audiences = Array.from(new Set([this.appId, legacyAudience].filter(Boolean)));
    this.apiUrl = options?.apiUrl || process.env.AUTH_API_URL || 'http://localhost:8080';
    this.issuer = options?.issuer || process.env.AUTH_ISSUER || 'https://auth.yourplatform.com';

    if (!this.appId) {
      throw new Error('AUTH_APP_ID is required. Pass it in options or set AUTH_APP_ID / AUTH_PUBLISHABLE_KEY to your publishable app id.');
    }

    this.jwksUrl = new URL(`${this.apiUrl}/api/v1/apps/${this.appId}/jwks`);
    this.jwks = jose.createRemoteJWKSet(this.jwksUrl);
  }

  async verifyToken(token: string): Promise<AuthPayload> {
    try {
      let lastError: Error | null = null;

      for (const audience of this.audiences) {
        try {
          const { payload } = await jose.jwtVerify(token, this.jwks, {
            issuer: this.issuer,
            audience,
          });

          const aud = Array.isArray(payload.aud) ? payload.aud[0] : payload.aud;

          return {
            userId: payload.sub as string,
            appId: aud as string,
            email: payload.email as string,
            ...payload,
          };
        } catch (error: any) {
          lastError = error;
        }
      }

      throw lastError || new Error('Token verification failed');
    } catch (error: any) {
      throw new Error(`Token verification failed: ${error.message}`);
    }
  }
}
