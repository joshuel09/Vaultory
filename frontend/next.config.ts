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
    return [
      {
        /*
         * Everything under /api goes to the Go service — except /api/auth, which is Better Auth's
         * own handler in this application.
         *
         * Without the exclusion the rewrite wins and registration reaches Go, which answers 404
         * because it has never served those routes. The negative lookahead is doing real work
         * here; it is not defensive decoration.
         */
        source: '/api/:path((?!auth$|auth/).*)',
        destination: `${backendOrigin}/api/:path`,
      },
    ]
  },
  experimental: {
    // A 10 MB upload over a slow connection needs time as well as room.
    proxyTimeout: 60_000,
    /*
     * The cap on a request body Next will carry through a rewrite (issue #47).
     *
     * Next's default is exactly 10 MiB, and it does not refuse a larger body — it **truncates** it
     * at the cap and leaves the stream open. The backend then waits for the rest of a
     * Content-Length that will never arrive, until proxyTimeout fires sixty seconds later and the
     * client gets a 500. Measured with a probe server behind the rewrite: send 12,000,000 bytes,
     * receive exactly 10,485,760, then an abort at 60s.
     *
     * 10 MiB is also exactly FR-010's own limit, so the two collided: a *legitimate* upload at the
     * top of the allowed range was broken in the same way as an oversized one, because the
     * multipart envelope pushed it over.
     *
     * The invariant this value has to satisfy: **the proxy must carry more than the backend will
     * ever accept**, so that the backend's answer is always what the client sees. Go refuses above
     * MaxUploadBytes + multipartSlack = 11,534,336 bytes, in images.go. 12 MiB clears that by a
     * megabyte.
     *
     * A body beyond this cap still truncates and still ends in a slow 500. That is unreachable
     * through the product — the browser refuses before sending, and Go refuses anything over
     * 11.5 MB long before this — so what remains is a hand-made request far outside any legitimate
     * use. Raising this number does not remove that cliff; it only moves it.
     *
     * An earlier comment here claimed Next streams a rewritten body through with no limit of its
     * own. That was wrong, and it is why nobody went looking for a cap.
     */
    middlewareClientMaxBodySize: '12mb',
  },
  images: {
    // Renditions are already 800x1000 and served from an authorized path; the optimizer would
    // strip the session and re-fetch server-side. See research.md Decision 2.
    unoptimized: true,
  },
}

export default nextConfig
