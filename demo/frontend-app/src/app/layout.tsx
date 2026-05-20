import type { ReactNode } from 'react';
import './globals.css';

export const metadata = {
  title: 'AuthService Demo',
  description: 'Demo app using one publishable key',
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
