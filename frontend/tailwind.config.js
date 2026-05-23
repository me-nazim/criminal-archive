/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        // Bengali primary, English fallback. The Bengali Unicode range
        // is served by Hind Siliguri / Noto; Latin glyphs fall through
        // to Inter for a cohesive bilingual feel.
        sans: [
          '"Hind Siliguri"',
          '"Noto Sans Bengali"',
          'Inter',
          'system-ui',
          'sans-serif',
        ],
        display: [
          '"Tiro Bangla"',
          '"Noto Serif Bengali"',
          '"Source Serif 4"',
          'serif',
        ],
        mono: ['"JetBrains Mono"', 'ui-monospace', 'SFMono-Regular', 'monospace'],
      },
      colors: {
        ink: {
          50: '#f6f7f9',
          100: '#eceef2',
          200: '#d5dae3',
          300: '#aeb6c4',
          400: '#828ea2',
          500: '#637087',
          600: '#4d586d',
          700: '#3f4859',
          800: '#363d4b',
          900: '#0f1320',
          950: '#080b15',
        },
        brand: {
          50: '#fff5f0',
          100: '#ffe6d8',
          200: '#ffc4a8',
          300: '#ff9a72',
          400: '#ff6e3d',
          500: '#e8501f',
          600: '#c63d14',
          700: '#9d3013',
          800: '#7c2913',
          900: '#5b1d0d',
        },
        accent: {
          50: '#f0f7ff',
          100: '#e0eefe',
          200: '#bedffd',
          300: '#8bc6fb',
          400: '#52a4f6',
          500: '#2c83eb',
          600: '#1d66c8',
          700: '#1a52a2',
          800: '#1b4684',
          900: '#1c3c6e',
        },
      },
      boxShadow: {
        soft: '0 1px 2px rgba(15,19,32,0.04), 0 4px 16px rgba(15,19,32,0.06)',
        elevated: '0 8px 24px rgba(15,19,32,0.08), 0 2px 6px rgba(15,19,32,0.04)',
      },
      borderRadius: {
        xl: '0.875rem',
        '2xl': '1.125rem',
      },
      backgroundImage: {
        'grid-light':
          'radial-gradient(circle at 1px 1px, rgba(15,19,32,0.05) 1px, transparent 0)',
        'hero-fade':
          'linear-gradient(180deg, rgba(255,245,240,0.6) 0%, rgba(246,247,249,0) 100%)',
      },
      keyframes: {
        'fade-in': {
          '0%': { opacity: 0, transform: 'translateY(4px)' },
          '100%': { opacity: 1, transform: 'translateY(0)' },
        },
      },
      animation: {
        'fade-in': 'fade-in 0.3s ease-out both',
      },
    },
  },
  plugins: [],
}
