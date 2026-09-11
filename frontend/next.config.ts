import type { NextConfig } from 'next'

/**
 * The browser talks only to this origin; /api/* is proxied to the Go service.
 *
 * This is what makes per-request image authorization work (FR-015): an <img> pointing at a
 * different origin would not carry the session cookie, and Next's image optimizer would fetch
 * server-side without the collector's credentials. See research.md Decision 2.
 */
const backendOrigin = process.env.VAULTORY_BACKEND_ORIGIN ?? 'http://127.0.0.1:8080'

const nextConfig: NextConfig = {
  reactStrictMode: true,
  async rewrites() {
    return [{ source: '/api/:path*', destination: `${backendOrigin}/api/:path*` }]
  },
  experimental: {
    // Uploads are capped at 10 MB by the backend (FR-010). The proxy limit must sit above that,
    // or an oversized upload is refused here and never reaches the rule that states the limit.
    proxyTimeout: 60_000,
  },
  images: {
    // Renditions are already 800x1000 and served from an authorized path; the optimizer would
    // strip the session and re-fetch server-side. See research.md Decision 2.
    unoptimized: true,
  },
}

export default nextConfig
