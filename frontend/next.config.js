/** @type {import('next').NextConfig} */
const nextConfig = {
  // Core Next.js flags
  reactStrictMode: true,
  swcMinify: true,

  // ---------------------------------------------------------------------------
  // Image handling – only allow domains that host document thumbnails / avatars.
  // The list can be supplied via a comma‑separated env var for flexibility.
  // ---------------------------------------------------------------------------
  images: {
    domains: (() => {
      const env = process.env.NEXT_PUBLIC_IMAGE_DOMAINS;
      if (!env) {
        // Default to localhost for local development; production should set this.
        return ['localhost'];
      }
      return env.split(',').map((d) => d.trim()).filter(Boolean);
    })(),
  },

  // ---------------------------------------------------------------------------
  // Expose only the variables that the client needs.
  // All names are prefixed with NEXT_PUBLIC_ so Next.js automatically injects them.
  // ---------------------------------------------------------------------------
  env: {
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL,
    NEXT_PUBLIC_OIDC_ISSUER: process.env.NEXT_PUBLIC_OIDC_ISSUER,
    NEXT_PUBLIC_OIDC_CLIENT_ID: process.env.NEXT_PUBLIC_OIDC_CLIENT_ID,
    NEXT_PUBLIC_OIDC_REDIRECT_URI: process.env.NEXT_PUBLIC_OIDC_REDIRECT_URI,
  },

  // ---------------------------------------------------------------------------
  // Global redirects – keep the entry point tidy.
  // ---------------------------------------------------------------------------
  async redirects() {
    return [
      {
        source: '/',
        destination: '/dashboard',
        permanent: true,
      },
    ];
  },

  // ---------------------------------------------------------------------------
  // Proxy API calls to the backend service.
  // This avoids CORS issues in the browser and lets us keep the same origin.
  // ---------------------------------------------------------------------------
  async rewrites() {
    const rawUrl = process.env.NEXT_PUBLIC_API_URL;
    if (!rawUrl) {
      // In CI or static export scenarios the API may not be reachable.
      console.warn(
        '[next.config] NEXT_PUBLIC_API_URL is not defined – API rewrites are disabled.'
      );
      return [];
    }

    // Strip any trailing slash to avoid double slashes in the final URL.
    const apiBase = rawUrl.replace(/\/+$/, '');

    return [
      {
        source: '/api/:path*',
        destination: `${apiBase}/api/:path*`,
      },
    ];
  },

  // ---------------------------------------------------------------------------
  // Extend webpack to support importing SVGs as React components.
  // ---------------------------------------------------------------------------
  webpack(config) {
    config.module.rules.push({
      test: /\.svg$/i,
      issuer: /\.[jt]sx?$/,
      use: ['@svgr/webpack'],
    });
    return config;
  },

  // ---------------------------------------------------------------------------
  // Security‑focused HTTP headers (optional but recommended for production).
  // ---------------------------------------------------------------------------
  async headers() {
    return [
      {
        source: '/(.*)',
        headers: [
          {
            key: 'X-Content-Type-Options',
            value: 'nosniff',
          },
          {
            key: 'X-Frame-Options',
            value: 'DENY',
          },
          {
            key: 'Referrer-Policy',
            value: 'strict-origin-when-cross-origin',
          },
          {
            key: 'Permissions-Policy',
            value: 'camera=(), microphone=(), geolocation=()',
          },
        ],
      },
    ];
  },
};

module.exports = nextConfig;