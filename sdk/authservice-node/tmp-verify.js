const { AuthVerifier } = require('./dist');

async function main() {
  const token = process.env.TEST_ACCESS_TOKEN;
  const apiUrl = process.env.TEST_API_URL;
  const appId = process.env.TEST_APP_ID;
  const audience = process.env.TEST_AUDIENCE;
  const issuer = process.env.TEST_ISSUER;

  const verifier = new AuthVerifier({ appId, audience, apiUrl, issuer });
  const payload = await verifier.verifyToken(token);
  console.log('POSITIVE_OK', JSON.stringify(payload));

  try {
    const wrongVerifier = new AuthVerifier({ appId, audience: '00000000-0000-0000-0000-000000000000', apiUrl, issuer });
    const wrongPayload = await wrongVerifier.verifyToken(token);
    console.log('NEGATIVE_UNEXPECTED_SUCCESS', JSON.stringify(wrongPayload));
  } catch (error) {
    console.log('NEGATIVE_ERROR', String(error && error.message || error));
  }
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
