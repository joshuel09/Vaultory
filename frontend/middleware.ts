import { NextResponse, type NextRequest } from 'next/server'

/**
 * Send a signed-out visitor to sign-in rather than into an error (FR-022, feature 004).
 *
 * This is a *navigation* decision, not an authorization one. It checks only that a session cookie
 * is present — it does not verify it, and it must not be mistaken for a guard. Authorization
 * happens in the Go service, which verifies every session itself and refuses anything it cannot
 * (FR-025). Somebody who forges a cookie gets past this redirect and straight into a 401.
 *
 * It exists because the alternative is what shipped: a vault page that renders "your collection
 * could not be loaded" to somebody who is simply not signed in, offering a "try again" button that
 * can never succeed.
 */
const SESSION_COOKIES = ['better-auth.session_token', '__Secure-better-auth.session_token']

export function middleware(request: NextRequest) {
  const signedIn = SESSION_COOKIES.some((name) => request.cookies.has(name))
  if (signedIn) return NextResponse.next()

  const next = request.nextUrl.pathname + request.nextUrl.search
  const url = new URL('/sign-in', request.url)
  url.searchParams.set('next', next)
  return NextResponse.redirect(url)
}

export const config = {
  // Vault pages only. The landing page, the auth pages and the API are all reachable signed out —
  // the API because Go answers 401 there, which is the correct response to a request rather than a
  // navigation.
  matcher: ['/collection/:path*'],
}
