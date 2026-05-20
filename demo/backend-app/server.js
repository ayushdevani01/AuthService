import 'dotenv/config';
import cors from 'cors';
import express from 'express';
import { requireAuth } from 'authservice-node';

const app = express();
const port = Number(process.env.PORT || 4000);

app.use(cors());
app.use(express.json());

const authOptions = {
  appId: process.env.AUTH_APP_ID || process.env.AUTH_PUBLISHABLE_KEY,
  apiUrl: process.env.AUTH_API_URL || 'http://localhost:8080',
  issuer: process.env.AUTH_ISSUER || 'https://auth.yourplatform.com',
};

app.get('/health', (_req, res) => {
  res.json({ ok: true });
});

app.get('/me', requireAuth(authOptions), (req, res) => {
  res.json({ user: req.auth });
});

if (!authOptions.appId) {
  console.error('Set AUTH_APP_ID (publishable key) in .env before starting the demo backend.');
  process.exit(1);
}

app.listen(port, () => {
  console.log(`Demo backend listening on http://localhost:${port}`);
  console.log(`Using publishable key ${authOptions.appId}`);
});
