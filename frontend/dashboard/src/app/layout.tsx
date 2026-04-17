import type { Metadata } from 'next';
import { Manrope, Source_Code_Pro } from 'next/font/google';
import { Toaster } from 'react-hot-toast';
import { AuthHydrator } from '@/components/auth-hydrator';
import { ThemeProvider } from '@/components/theme-provider';
import './globals.css';

const manrope = Manrope({
  subsets: ['latin'],
  variable: '--font-sans',
  display: 'swap',
});

const sourceCodePro = Source_Code_Pro({
  subsets: ['latin'],
  variable: '--font-mono',
  display: 'swap',
});

export const metadata: Metadata = {
  title: 'AuthService Dashboard',
  description: 'Developer portal for AuthService',
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body className={`${manrope.variable} ${sourceCodePro.variable}`}>
        <ThemeProvider>
          <AuthHydrator />
          {children}
        </ThemeProvider>
        <Toaster
          position="top-right"
          toastOptions={{
            style: {
              background: 'var(--panel)',
              color: 'var(--foreground)',
              border: '1px solid var(--border)',
            },
          }}
        />
      </body>
    </html>
  );
}
