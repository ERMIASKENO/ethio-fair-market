import type { NextConfig } from 'next';

// Local development without Kubernetes: DEV_API_PROXY=1 forwards /v1/* to locally running services.
// In the cluster the HTTPS gateway routes /v1/<service> directly, so no rewrites are needed.
const devTargets: Record<string, string> = {
  auth: process.env.IDENTITY_URL ?? 'http://localhost:8081',
  listings: process.env.LISTINGS_URL ?? 'http://localhost:8082',
  categories: process.env.CATALOG_URL ?? 'http://localhost:8083',
  locations: process.env.LOCATIONS_URL ?? 'http://localhost:8084',
};

const config: NextConfig = {
  output: 'standalone', // minimal server bundle for the container image
  poweredByHeader: false,
  reactStrictMode: true,
  images: { unoptimized: true }, // media service (phase 2) will serve resized images
  async headers() {
    return [{
      source: '/:path*',
      headers: [
        { key: 'X-Content-Type-Options', value: 'nosniff' },
        { key: 'Referrer-Policy', value: 'strict-origin-when-cross-origin' },
        { key: 'X-Frame-Options', value: 'DENY' },
        { key: 'Permissions-Policy', value: 'camera=(), microphone=(), geolocation=()' },
      ],
    }];
  },
  async rewrites() {
    if (process.env.DEV_API_PROXY !== '1') return [];
    return Object.entries(devTargets).map(([prefix, target]) => ({ source: `/v1/${prefix}/:path*`, destination: `${target}/v1/${prefix}/:path*` }))
      .concat(Object.entries(devTargets).map(([prefix, target]) => ({ source: `/v1/${prefix}`, destination: `${target}/v1/${prefix}` })));
  },
};

export default config;
