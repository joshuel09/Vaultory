import Link from 'next/link'
import { cookies } from 'next/headers'
import { CollectionGallery } from '@/components/collection/CollectionGallery'
import { StatusFilter } from '@/components/collection/StatusFilter'
import { EmptyCollection } from '@/components/collection/EmptyCollection'
import { NoResults } from '@/components/collection/NoResults'
import { buttonClasses } from '@/components/ui/button'
import { isCollectionStatus, type CollectionPage, type CollectionStatus } from '@/lib/api/types'

/**
 * The collection view. A Server Component: the first page is rendered on the server, and only the
 * gallery's incremental loading and the filter need to be interactive (Technology Constraints:
 * prefer RSC, Client Components only where browser state requires it).
 */

// A collection is per-collector and changes as they add to it.
export const dynamic = 'force-dynamic'

const BACKEND = process.env.VAULTORY_BACKEND_ORIGIN ?? 'http://127.0.0.1:8080'

async function fetchFirstPage(status: CollectionStatus | null): Promise<CollectionPage> {
  // Server-side, the rewrite is not in play, so this goes to the backend directly — and must
  // forward the collector's cookie itself, or the request arrives with no identity and is refused
  // (FR-029). Browser-side requests carry it automatically via the same-origin rewrite.
  const cookieHeader = (await cookies()).toString()
  const url = new URL('/api/collectibles', BACKEND)
  if (status) url.searchParams.set('status', status)

  const response = await fetch(url, {
    headers: cookieHeader ? { cookie: cookieHeader } : {},
    cache: 'no-store',
  })
  if (!response.ok) {
    // Thrown so the route's error boundary renders the designed error state (FR-043).
    throw new Error(`collection request failed with ${response.status}`)
  }
  return (await response.json()) as CollectionPage
}

export default async function CollectionPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>
}) {
  const params = await searchParams
  const raw = typeof params.status === 'string' ? params.status : ''
  const status = isCollectionStatus(raw) ? raw : null

  const page = await fetchFirstPage(status)
  const isEmptyVault = page.totalUnfiltered === 0
  const filterMatchedNothing = !isEmptyVault && page.items.length === 0 && status !== null

  return (
    <main className="mx-auto w-full max-w-[110rem] px-4 py-8 sm:px-6 lg:px-10">
      <header className="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-ink">Your vault</h1>
          <p className="mt-1 text-sm text-ink-muted">
            {page.totalUnfiltered === 0
              ? 'Nothing here yet'
              : `${page.totalUnfiltered} ${page.totalUnfiltered === 1 ? 'collectible' : 'collectibles'}`}
          </p>
        </div>
        <Link href="/collection/new" className={buttonClasses()}>
          Add a collectible
        </Link>
      </header>

      {/* The filter is pointless on an empty vault, and showing it would imply there is something
          to narrow. */}
      {!isEmptyVault && (
        <div className="mb-6">
          <StatusFilter active={status} />
        </div>
      )}

      {isEmptyVault ? (
        <EmptyCollection />
      ) : filterMatchedNothing ? (
        <NoResults status={status} total={page.totalUnfiltered} />
      ) : (
        <CollectionGallery
          // A new filter is a new collection: remount rather than reconcile, so no item from the
          // previous filter can survive into this one.
          key={status ?? 'all'}
          initialItems={page.items}
          initialCursor={page.nextCursor ?? null}
          status={status}
        />
      )}
    </main>
  )
}
