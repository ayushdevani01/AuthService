import type { Config } from 'tailwindcss';

const config: Config = {
  content: ['./src/**/*.{js,ts,jsx,tsx,mdx}'],
  theme: {
    extend: {
      colors: {
        obsidian: '#020202',
        night: '#050505',
        ash: '#111111',
        graphite: '#171717',
        slate: '#262626',
        mist: '#a3a3a3',
      },
      boxShadow: {
        luxe: '0 24px 80px rgba(255,255,255,0.06)',
        insetGlow: 'inset 0 1px 0 rgba(255,255,255,0.08)',
      },
      backgroundImage: {
        noise: 'radial-gradient(circle at top, rgba(255,255,255,0.1), transparent 35%), linear-gradient(180deg, #050505 0%, #020202 100%)',
      },
      borderRadius: {
        xl2: '1.5rem',
      },
    },
  },
  plugins: [],
};

export default config;
