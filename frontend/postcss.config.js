/** @type {import('postcss').ProcessOptions} */
const isProduction = process.env.NODE_ENV === 'production';

module.exports = {
  // PostCSS plugins are applied in the order they appear.
  plugins: {
    // Tailwind CSS utilities and components.
    tailwindcss: {},

    // Adds vendor prefixes for cross‑browser compatibility.
    autoprefixer: {},

    // Minify CSS in production builds for optimal payload size.
    ...(isProduction
      ? {
          cssnano: {
            preset: 'default',
          },
        }
      : {}),
  },
};