import Link from 'next/link'
import { cookies } from 'next/headers'
import { redirect } from 'next/navigation'
import { CollectionGallery } from '@/components/collection/CollectionGallery'
import { StatusFilter } from '@/components/collection/StatusFilter'
import { EmptyCollection } from '@/components/collection/EmptyCollection'
import { NoResults } from '@/components/collection/NoResults'
import { UndoDeleteNotice } from '@/components/collection/UndoDeleteNotice'
import { buttonClasses } from '@/components/ui/button'
import { isCollectionStatus, type CollectionPage } from '@/lib/api/types'
import { galleryHref, isFiltered, type GalleryFilter } from '@/lib/gallery-link'

/**
 * The collection view. A Server Component: the first page is rendered on the server, and only the
 * gallery's incremental loading and the filter need to be interactive (Technology Constraints:
 * prefer RSC, Client Components only where browser state requires it).
 */

// A collection is per-collector and changes as they add to it.
export const dynamic = 'force-dynamic'

const BACKEND = process.env.VAULTORY_BACKEND_ORIGIN ?? 'http://127.0.0.1:8080'

async function fetchFirstPage(filter: GalleryFilter): Promise<CollectionPage> {
  // Server-side, the rewrite is not in play, so this goes to the backend directly — and must
  // forward the collector's cookie itself, or the request arrives with no identity and is refused
  // (FR-029). Browser-side requests carry it automatically via the same-origin rewrite.
  const cookieHeader = (await cookies()).toString()
  const url = new URL('/api/collectibles', BACKEND)
  // Null means no filter; the empty string is a filter — the pieces recording no series at all.
  // Truthiness cannot tell the two apart, and getting it wrong here makes the statistics'
  // unspecified group link to the whole vault (008 FR-031).
  if (filter.status != null) url.searchParams.set('status', filter.status)
  if (filter.series != null) url.searchParams.set('series', filter.series)
  if (filter.manufacturer != null) url.searchParams.set('manufacturer', filter.manufacturer)

  const response = await fetch(url, {
    headers: cookieHeader ? { cookie: cookieHeader } : {},
    cache: 'no-store',
  })
  if (response.status === 401) {
    /*
     * Not signed in — which is not the same as unreachable, and must not look like it.
     *
     * The middleware only checks that a session cookie is present, so a cookie that has been
     * revoked gets this far: a collector whose password was reset on another device arrives here
     * holding a cookie that Go no longer accepts. Without this they would see "your collection
     * could not be loaded" and a Try again button that can never succeed — the exact state
     * FR-022 exists to eliminate.
     */
    redirect('/sign-in?next=/collection')
  }
  if (!response.ok) {
    // A genuine failure to reach the collection. Thrown so the route's error boundary renders the
    // designed state (FR-043), which now means what it says.
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

  // typeof === 'string', not truthiness: `?series=` arrives as '' and is a real filter, while an
  // absent parameter arrives as undefined and is not.
  const filter: GalleryFilter = {
    status,
    series: typeof params.series === 'string' ? params.series : null,
    manufacturer: typeof params.manufacturer === 'string' ? params.manufacturer : null,
  }

  // Set by the edit screen after a deletion, so the way back is offered where the collector lands
  // (feature 010, FR-005). Only a well-formed id is acted on; anything else is ignored.
  const deleted =
    typeof params.deleted === 'string' && /^[0-9a-f-]{36}$/i.test(params.deleted) ? params.deleted : null

  const page = await fetchFirstPage(filter)
  const isEmptyVault = page.totalUnfiltered === 0
  const filterMatchedNothing = !isEmptyVault && page.items.length === 0 && isFiltered(filter)

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
        <div className="flex flex-wrap items-center gap-3">
          {/* Only offered once there is something to summarise; a statistics link on an empty vault
              leads to a designed apology (FR-029). */}
          {!isEmptyVault && (
            <Link
              href="/collection/stats"
              data-testid="statistics-link"
              className="text-sm font-medium text-ink-muted underline-offset-4 hover:text-ink hover:underline"
            >
              What your collection holds
            </Link>
          )}
          <Link href="/collection/new" className={buttonClasses()}>
            Add a collectible
          </Link>
        </div>
      </header>

      {deleted && <UndoDeleteNotice deletedId={deleted} galleryHref={galleryHref(filter)} />}

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
        <NoResults filter={filter} total={page.totalUnfiltered} />
      ) : (
        <CollectionGallery
          // A new filter is a new collection: remount rather than reconcile, so no item from the
          // previous filter can survive into this one. The key covers every filter, not just the
          // status — keying on one of three means switching series reconciles into stale items.
          key={galleryHref(filter)}
          initialItems={page.items}
          initialCursor={page.nextCursor ?? null}
          filter={filter}
        />
      )}
    </main>
  )
}
