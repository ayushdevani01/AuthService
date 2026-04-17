import express from 'express';
import { requireAuth } from './src';

const app = express();
const PORT = process.env.PORT || 3001;

app.use(express.json());

const authOptions = {
  appId: process.env.AUTH_APP_ID || 'your-public-app-id',
  audience: process.env.AUTH_AUDIENCE,
  apiUrl: process.env.AUTH_API_URL || 'http://localhost:8080',
  issuer: process.env.AUTH_ISSUER || 'https://auth.yourplatform.com'
};

app.get('/', (_req, res) => {
  res.json({
    status: 'ok',
    message: 'AuthService Example Server',
    endpoints: {
      public: 'GET /',
      protected: 'GET /protected',
      verify: 'GET /whoami'
    }
  });
});

app.get('/protected', requireAuth(authOptions), (req, res) => {
  res.json({
    message: 'You have access!',
    user: req.auth
  });
});

app.get('/whoami', requireAuth(authOptions), (req, res) => {
  res.json(req.auth);
});

app.listen(PORT, () => {
  console.log(`\nAuthService Example Server running on http://localhost:${PORT}`);
  console.log(`\nEndpoints:`);
  console.log(`   GET /          - Public health check`);
  console.log(`   GET /protected - Protected endpoint (requires valid JWT)`);
  console.log(`   GET /whoami    - Get current user info from token`);
  console.log(`\nEnvironment variables:`);
  console.log(`   AUTH_APP_ID - Your public app ID used for JWKS lookup`);
  console.log(`   AUTH_AUDIENCE - Your internal app UUID used as the JWT aud value`);
  console.log(`   AUTH_API_URL - AuthService API URL (default: http://localhost:8080)`);
  console.log(`   AUTH_ISSUER - Token issuer (default: https://auth.yourplatform.com)`);
  console.log(`\nUsage: Set AUTH_APP_ID and AUTH_AUDIENCE, then make requests with 'Authorization: Bearer <jwt-token>'`);
});
