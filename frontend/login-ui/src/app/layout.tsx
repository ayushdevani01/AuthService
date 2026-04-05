import type { Metadata } from 'next';
import { Toaster } from 'react-hot-toast';
import { ThemeProvider } from '@/components/theme-provider';
import './globals.css';

export const metadata: Metadata = {
  title: 'AuthService Hosted Login',
  description: 'Hosted login experience for AuthService apps',
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body>
        <ThemeProvider>{children}</ThemeProvider>
        <Toaster position="top-right" toastOptions={{ style: { background: 'var(--panel)', color: 'var(--foreground)', border: '1px solid var(--border)' } }} />
      </body>
    </html>
  );
}
