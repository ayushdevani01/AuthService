import type { Config } from 'tailwindcss';

const config: Config = {
  content: ['./src/**/*.{js,ts,jsx,tsx,mdx}'],
  theme: {
    extend: {
      colors: {
        obsidian: '#020202',
        night: '#050505',
        graphite: '#171717',
      },
      boxShadow: {
        luxe: '0 30px 90px rgba(255,255,255,0.06)',
      },
      borderRadius: {
        xl2: '1.75rem',
      },
    },
  },
  plugins: [],
};

export default config;
