module.exports = {
  darkMode: 'class', // Enable class-based dark mode toggling
  content: [
    './frontend/src/**/*.{js,jsx,ts,tsx}',
    './frontend/pages/**/*.{js,jsx,ts,tsx}',
    './frontend/components/**/*.{js,jsx,ts,tsx}',
    './frontend/app/**/*.{js,jsx,ts,tsx}',
  ],
  theme: {
    container: {
      center: true,
      padding: '1rem',
      screens: {
        sm: '640px',
        md: '768px',
        lg: '1024px',
        xl: '1280px',
        '2xl': '1536px',
      },
    },
    extend: {
      colors: {
        primary: '#0A74DA',
        secondary: '#14A44D',
        accent: '#F59E0B',
        neutral: {
          light: '#F5F7FA',
          dark: '#1F2937',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        heading: ['Roboto Slab', 'serif'],
      },
      borderRadius: {
        DEFAULT: '0.5rem',
      },
      // The default spacing scale already uses a 4‑px base (1 = 0.25rem),
      // but we expose a few convenient aliases that match the design system.
      spacing: {
        1: '0.25rem', // 4px
        2: '0.5rem',  // 8px
        3: '0.75rem', // 12px
        4: '1rem',    // 16px
        5: '1.25rem', // 20px
        6: '1.5rem',  // 24px
        8: '2rem',    // 32px
        10: '2.5rem', // 40px
        12: '3rem',   // 48px
        16: '4rem',   // 64px
        20: '5rem',   // 80px
        24: '6rem',   // 96px
        32: '8rem',   // 128px
        40: '10rem',  // 160px
        48: '12rem',  // 192px
        56: '14rem',  // 224px
        64: '16rem',  // 256px
      },
    },
  },
  plugins: [
    require('@tailwindcss/forms'),       // Better default styling for form elements
    require('@tailwindcss/typography'), // Prose utilities for rich text
    require('@tailwindcss/aspect-ratio'),// Consistent aspect‑ratio handling
    require('@tailwindcss/line-clamp'), // Truncate text with line‑clamp
  ],
};