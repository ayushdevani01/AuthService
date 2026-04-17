import { Request, Response, NextFunction } from 'express';
import { AuthVerifier, AuthOptions, AuthPayload } from './verifier';

// Augment the Express Request type
declare global {
  namespace Express {
    interface Request {
      auth?: AuthPayload;
    }
  }
}

export const requireAuth = (options?: AuthOptions) => {
  const verifier = new AuthVerifier(options);

  return async (req: Request, res: Response, next: NextFunction): Promise<void> => {
    try {
      const authHeader = req.headers.authorization;
      if (!authHeader || !authHeader.startsWith('Bearer ')) {
        res.status(401).json({ error: 'Missing or invalid authentication token' });
        return;
      }

      const token = authHeader.split(' ')[1];
      const payload = await verifier.verifyToken(token);
      
      req.auth = payload;
      next();
    } catch (error: any) {
      res.status(401).json({ error: error.message });
      return;
    }
  };
};

export { AuthVerifier, AuthOptions, AuthPayload } from './verifier';
