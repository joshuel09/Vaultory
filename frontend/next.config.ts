import type { NextConfig } from 'next'

/**
 * The browser talks only to this origin; /api/* is proxied to the Go service.
 *
 * This is what makes per-request image authorization work (FR-015): an <img> pointing at a
 * different origin would not carry the session cookie, and Next's image optimizer would fetch
 * server-side without the collector's credentials. See research.md Decision 2.
 *
 * Read at BUILD time, not at start. Next bakes rewrites into the build manifest, so setting this
 * on a running production container does nothing — a class of bug that works in `next dev` and
 * fails silently once built. The production image passes it as a build argument for that reason.
 */
const backendOrigin = process.env.VAULTORY_BACKEND_ORIGIN ?? 'http://127.0.0.1:8080'

const nextConfig: NextConfig = {
  reactStrictMode: true,
  /**
   * Emits .next/standalone: a minimal server.js beside only the node_modules it actually reached.
   * It is what lets the production image ship without the dependency tree or the build toolchain
   * (FR-017), and it is inert for `next dev`, so this costs the development path nothing.
   */
  output: 'standalone',
  async rewrites() {
    return [{ source: '/api/:path*', destination: `${backendOrigin}/api/:path*` }]
  },
  experimental: {
    // A 10 MB upload over a slow connection needs time, not a larger cap: Next streams a rewritten
    // request body through without a size limit of its own, so there is no body cap here to raise.
    // The only limit is the backend's, which is the rule that states it (FR-010).
    proxyTimeout: 60_000,
  },
  images: {
    // Renditions are already 800x1000 and served from an authorized path; the optimizer would
    // strip the session and re-fetch server-side. See research.md Decision 2.
    unoptimized: true,
  },
}

export default nextConfig
