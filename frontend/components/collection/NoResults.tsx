import { STATUS_LABELS, type CollectionStatus } from '@/lib/api/types'

/**
 * A filter that matched nothing (FR-040).
 *
 * Deliberately different from the empty-collection state in wording, shape, and the action it
 * offers: the collector's vault is not empty, this slice of it is. Telling them "your vault is
 * empty" here would be a lie.
 */
export function NoResults({ status, total }: { status: CollectionStatus; total: number }) {
  return (
    <div
      data-testid="no-results"
      className="flex flex-col items-center justify-center rounded-card border border-edge bg-surface px-6 py-16 text-center"
    >
      <span aria-hidden="true" className="mb-4 text-3xl text-ink-faint/60">
        ⌕
      </span>
      <h2 className="text-base font-medium text-ink">
        Nothing marked {STATUS_LABELS[status]}
      </h2>
      <p className="mt-2 max-w-sm text-sm text-ink-muted">
        Your vault holds {total} {total === 1 ? 'collectible' : 'collectibles'}, but none with this
        status. Choose <span className="text-ink">All</span> to see everything.
      </p>
    </div>
  )
}
