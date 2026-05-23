/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      fontFamily: {
        // Bengali primary, English fallback
        sans: [
          '"Hind Siliguri"',
          '"Noto Sans Bengali"',
          'Inter',
          'system-ui',
          'sans-serif',
        ],
        display: ['"Tiro Bangla"', '"Noto Serif Bengali"', 'serif'],
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
      },
    },
  },
  plugins: [],
}
