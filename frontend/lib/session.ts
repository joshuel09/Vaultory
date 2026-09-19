import { headers } from 'next/headers'
import { auth } from '@/lib/auth'

/**
 * The signed-in collector, or null.
 *
 * Read server-side so a page can render the right thing on the first paint rather than flashing a
 * "Sign in" link at somebody who already is. This answers "who is looking at this page" for
 * presentation only — it never decides what a collector may reach. That remains the Go service's
 * job, on every request, and this value is not sent to it.
 */
export async function currentCollector() {
  const session = await auth.api.getSession({ headers: await headers() })
  return session?.user ?? null
}
