import Link from 'next/link'
import { cookies } from 'next/headers'
import { notFound, redirect } from 'next/navigation'
import { EditCollectibleForm } from '@/components/collection/EditCollectibleForm'
import { safeRedirect } from '@/lib/safe-redirect'
import type { Collectible } from '@/lib/api/types'

/**
 * The edit screen. A Server Component, like the gallery: the collectible is fetched and rendered on
 * the server, and only the form itself needs to be interactive.
 */

// A collectible changes as its collector edits it, and a stale copy is exactly what FR-027 exists
// to refuse. Never serve a cached one.
export const dynamic = 'force-dynamic'

const BACKEND = process.env.VAULTORY_BACKEND_ORIGIN ?? 'http://127.0.0.1:8080'

async function fetchCollectible(id: string): Promise<Collectible> {
  // Server-side, the Next.js rewrite is not in play, so this goes to the Go service directly — and
  // must forward the collector's cookie itself, or the request arrives with no identity and is
  // refused (FR-033). Browser-side requests carry it automatically through the same-origin rewrite.
  const cookieHeader = (await cookies()).toString()
  const response = await fetch(new URL(`/api/collectibles/${encodeURIComponent(id)}`, BACKEND), {
    headers: cookieHeader ? { cookie: cookieHeader } : {},
    cache: 'no-store',
  })

  if (response.status === 401) {
    /*
     * Not signed in — which is not the same as unreachable, and must not look like it.
     *
     * The middleware only checks that a session cookie is present, so a cookie that has been
     * revoked gets this far: a collector whose password was reset on another device arrives here
     * holding a cookie Go no longer accepts.
     */
    redirect(`/sign-in?next=/collection/${encodeURIComponent(id)}/edit`)
  }
  if (response.status === 404) {
    // Gone, or never theirs — the two are deliberately indistinguishable (FR-031). Either way the
    // not-found state is the honest one; an error state with a Try again button would be a lie,
    // because retrying can never succeed.
    notFound()
  }
  if (!response.ok) {
    // A genuine failure to reach the collectible. Thrown so the route's error boundary renders the
    // designed state, which now means what it says.
    throw new Error(`collectible request failed with ${response.status}`)
  }
  return (await response.json()) as Collectible
}

export default async function EditCollectiblePage({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>
  searchParams: Promise<Record<string, string | string[] | undefined>>
}) {
  const { id } = await params
  const query = await searchParams
  const collectible = await fetchCollectible(id)

  // Where to send the collector afterwards. Carried from the gallery so a collector who was
  // browsing one status filter is returned to it rather than dropped on an unfiltered gallery,
  // and constrained to this site's own paths by the same guard sign-in uses.
  const returnTo = safeRedirect(typeof query.next === 'string' ? query.next : null)

  return (
    <div className="mx-auto w-full max-w-3xl px-4 py-10 sm:px-6">
      <nav className="mb-8">
        <Link
          href={returnTo}
          className="text-sm text-ink-muted underline-offset-4 hover:text-ink hover:underline"
        >
          ← Back to my collection
        </Link>
      </nav>

      <header className="mb-10 space-y-2">
        <h1 className="text-2xl font-semibold tracking-tight text-ink sm:text-3xl">
          Edit {collectible.name}
        </h1>
        <p className="text-sm text-ink-muted">
          Change anything here. Nothing moves in your gallery — a collectible keeps the place it has
          had since you added it.
        </p>
      </header>

      <EditCollectibleForm collectible={collectible} returnTo={returnTo} />
    </div>
  )
}
