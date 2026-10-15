import Link from 'next/link'
import { cookies } from 'next/headers'
import { redirect } from 'next/navigation'
import { CollectionSummary } from '@/components/collection/CollectionSummary'
import { StatusBreakdown } from '@/components/collection/StatusBreakdown'
import { GroupBreakdown } from '@/components/collection/GroupBreakdown'
import { buttonClasses } from '@/components/ui/button'
import type { CollectionStatistics } from '@/lib/api/types'

/**
 * The statistics screen, at its own route.
 *
 * Its own route rather than a strip above the gallery, so the aggregate queries stay off the page
 * collectors open most, and so the figures have room to be legible at mobile width (FR-028).
 */

// The figures describe a vault that changes as the collector edits it. Computed per request so
// there is nothing to go stale (FR-039).
export const dynamic = 'force-dynamic'

const BACKEND = process.env.VAULTORY_BACKEND_ORIGIN ?? 'http://127.0.0.1:8080'

async function fetchStatistics(): Promise<CollectionStatistics> {
  // Server-side the Next.js rewrite is not in play, so this reaches Go directly and must forward
  // the collector's cookie itself; without it the request arrives with no identity and is refused.
  const cookieHeader = (await cookies()).toString()
  const response = await fetch(new URL('/api/collectibles/statistics', BACKEND), {
    headers: cookieHeader ? { cookie: cookieHeader } : {},
    cache: 'no-store',
  })

  if (response.status === 401) {
    // Not signed in, which is not the same as unreachable. The middleware only checks that a
    // session cookie is present, so a revoked one gets this far (FR-032).
    redirect('/sign-in?next=/collection/stats')
  }
  if (!response.ok) {
    throw new Error(`statistics request failed with ${response.status}`)
  }
  return (await response.json()) as CollectionStatistics
}

export default async function CollectionStatisticsPage() {
  const stats = await fetchStatistics()

  return (
    <div className="mx-auto w-full max-w-4xl px-4 py-10 sm:px-6">
      <nav className="mb-8">
        <Link
          href="/collection"
          className="text-sm text-ink-muted underline-offset-4 hover:text-ink hover:underline"
        >
          ← Back to my collection
        </Link>
      </nav>

      <header className="mb-10 space-y-2">
        <h1 className="text-2xl font-semibold tracking-tight text-ink sm:text-3xl">
          What your collection holds
        </h1>
        <p className="text-sm text-ink-muted">
          Every figure here counts only your own vault.
        </p>
      </header>

      {stats.totalCount === 0 ? (
        /*
          An empty vault gets a designed state rather than a grid of zeros. Zeros are true and
          useless: they read as a broken screen, and they offer a collector nothing to do (FR-034).
        */
        <section
          data-testid="stats-empty"
          className="rounded-xl border border-dashed border-edge bg-surface-raised p-10 text-center"
        >
          <h2 className="text-lg font-medium text-ink">Nothing to summarise yet</h2>
          <p className="mx-auto mt-2 max-w-md text-sm text-ink-muted">
            Once your vault holds a collectible or two, this is where you will see what it holds and
            what you have spent on it.
          </p>
          <Link href="/collection/new" className={buttonClasses({ className: 'mt-6' })}>
            Add your first collectible
          </Link>
        </section>
      ) : (
        <div className="space-y-12">
          <CollectionSummary stats={stats} />
          <StatusBreakdown byStatus={stats.byStatus} />
          <GroupBreakdown
            dimension="series"
            groups={stats.bySeries}
            heading="Series you collect most"
            emptyNote="None of your pieces records a series yet."
          />
          <GroupBreakdown
            dimension="manufacturer"
            groups={stats.byManufacturer}
            heading="Manufacturers you buy from most"
            emptyNote="None of your pieces records a manufacturer yet."
          />
        </div>
      )}
    </div>
  )
}
