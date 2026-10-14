import type { CollectionStatistics, StatusSummary } from '@/lib/api/types'
import { formatMoney } from '@/lib/format-money'

/**
 * The headline figures: how many pieces the vault holds, and what was paid for the ones owned.
 *
 * Every number here arrives finished. This component formats and labels; it does not sum, count, or
 * compare anything, because a business rule living in the presentation layer is a bug (FR-001).
 */

function summaryFor(stats: CollectionStatistics, status: string): StatusSummary | undefined {
  // Not a derivation — a lookup. The backend always returns all four, in a fixed order.
  return stats.byStatus.find((s) => s.status === status)
}

export function CollectionSummary({ stats }: { stats: CollectionStatistics }) {
  const owned = summaryFor(stats, 'owned')
  const unpriced = stats.totalCount - stats.pricedCount

  return (
    <section data-testid="collection-summary" aria-labelledby="summary-heading" className="space-y-6">
      <h2 id="summary-heading" className="sr-only">
        What your collection holds
      </h2>

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="rounded-xl border border-edge bg-surface-raised p-6">
          <p className="text-sm font-medium text-ink-muted">Pieces in your vault</p>
          <p data-testid="total-count" className="mt-2 text-3xl font-semibold tracking-tight text-ink">
            {stats.totalCount}
          </p>
        </div>

        <div className="rounded-xl border border-edge bg-surface-raised p-6">
          {/*
            Owned alone, deliberately. Preordered money has not been spent, a wishlist is aspiration,
            and a sold piece is money that came back — adding all four produces a figure that
            describes nothing a collector can act on (FR-012).
          */}
          <p className="text-sm font-medium text-ink-muted">Paid for what you own</p>
          <p data-testid="owned-total" className="mt-2 text-3xl font-semibold tracking-tight text-ink">
            {formatMoney(owned?.total ?? '0.00')}
          </p>
        </div>
      </div>

      {unpriced > 0 && (
        <p data-testid="unpriced-note" className="text-sm text-ink-muted">
          {unpriced === stats.totalCount ? (
            <>
              None of your {stats.totalCount} pieces has a recorded price yet, so there is nothing to
              total. Add a purchase price to a collectible and it will appear here.
            </>
          ) : (
            <>
              {/*
                FR-011: a total that silently covers only part of the collection is worse than no
                total. Say how many pieces it misses, in a sentence a collector can act on.
              */}
              These totals cover the {stats.pricedCount} of {stats.totalCount} pieces with a recorded
              price. {unpriced} {unpriced === 1 ? 'piece has' : 'pieces have'} no price recorded.
            </>
          )}
        </p>
      )}
    </section>
  )
}
