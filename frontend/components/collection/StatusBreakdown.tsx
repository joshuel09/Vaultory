import Link from 'next/link'
import type { CollectionStatus, StatusSummary } from '@/lib/api/types'
import { STATUS_LABELS } from '@/lib/api/types'
import { formatMoney } from '@/lib/format-money'

/**
 * The four statuses, each with its own count and its own total.
 *
 * It iterates what it was given rather than looping over the four status names and looking each one
 * up. The difference matters: a lookup would invent a zero for a status the backend did not send,
 * which is the frontend answering a question about the vault. The contract fixes this array at
 * exactly four entries and the backend fills the empty ones (FR-015), so displaying the array as it
 * arrives is both complete and honest.
 */

/**
 * What each total is, said in words.
 *
 * Only Owned and Sold record money that actually left the collector's hands. A price on a wishlist
 * entry is what the piece costs, not what anyone paid, and calling it "paid" would be a small lie
 * that compounds into a wrong total. Sold says "paid" because FR-019 requires those words.
 */
const MONEY_LABELS: Record<CollectionStatus, string> = {
  owned: 'paid for these',
  preordered: 'recorded against these',
  wishlist: 'recorded against these',
  sold: 'paid for these',
}

export function StatusBreakdown({ byStatus }: { byStatus: StatusSummary[] }) {
  return (
    <section data-testid="status-breakdown" aria-labelledby="status-breakdown-heading">
      <h2
        id="status-breakdown-heading"
        className="text-lg font-medium tracking-tight text-ink"
      >
        Where your pieces stand
      </h2>
      <p className="mt-1 text-sm text-ink-muted">
        Each figure covers one status alone. Select one to see the pieces behind it.
      </p>

      <ul className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {byStatus.map((summary) => (
          <li key={summary.status}>
            {/* FR-030: a number a collector cannot trace back is a number they cannot trust. */}
            <Link
              href={`/collection?status=${summary.status}`}
              data-testid={`status-${summary.status}`}
              className="block h-full rounded-xl border border-edge bg-surface-raised p-5 transition-colors hover:border-ink-faint focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
            >
              <p className="text-sm font-medium text-ink-muted">
                {STATUS_LABELS[summary.status]}
              </p>
              <p
                data-testid={`status-${summary.status}-count`}
                className="mt-2 text-2xl font-semibold tracking-tight text-ink"
              >
                {summary.count}
                <span className="ml-1.5 text-sm font-normal text-ink-muted">
                  {summary.count === 1 ? 'piece' : 'pieces'}
                </span>
              </p>
              <p className="mt-1 text-sm text-ink-muted">
                <span data-testid={`status-${summary.status}-total`} className="text-ink">
                  {formatMoney(summary.total)}
                </span>{' '}
                {MONEY_LABELS[summary.status]}
              </p>
            </Link>

            {/*
              FR-019 again, and the part that is easy to leave out: nothing records what a piece
              sold for, so the figure above is what the collector spent, not what they got back. A
              reader who assumes otherwise reads a loss as a gain.
            */}
            {summary.status === 'sold' && (
              <p data-testid="sold-caveat" className="mt-2 px-1 text-xs text-ink-muted">
                What you paid for them. Nothing here records what they sold for.
              </p>
            )}
          </li>
        ))}
      </ul>
    </section>
  )
}
