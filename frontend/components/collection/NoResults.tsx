import Link from 'next/link'
import { STATUS_LABELS } from '@/lib/api/types'
import type { GalleryFilter } from '@/lib/gallery-link'

/**
 * A filter that matched nothing (FR-040).
 *
 * Deliberately different from the empty-collection state in wording, shape, and the action it
 * offers: the collector's vault is not empty, this slice of it is. Telling them "your vault is
 * empty" here would be a lie.
 *
 * It names the filter that matched nothing, including the series and manufacturer filters a
 * statistics link can arrive with. A group shown in the statistics always has pieces behind it, so
 * reaching this state through one of those links means the vault changed underneath — or that the
 * URL was edited. Either way, saying which filter came up empty is more use than a shrug.
 */

function describe(filter: GalleryFilter): string {
  // Against null and undefined, not truthiness: '' is the filter for pieces recording nothing, and
  // it is the one most likely to land here.
  const group =
    filter.series != null
      ? { kind: 'series', value: filter.series }
      : filter.manufacturer != null
        ? { kind: 'manufacturer', value: filter.manufacturer }
        : null

  if (group && filter.status != null) return 'Nothing matches both of those filters'
  if (group) {
    return group.value === ''
      ? `Nothing without a ${group.kind} recorded`
      : `Nothing in ${group.value}`
  }
  if (filter.status != null) return `Nothing marked ${STATUS_LABELS[filter.status]}`
  return 'Nothing matches this filter'
}

export function NoResults({ filter, total }: { filter: GalleryFilter; total: number }) {
  return (
    <div
      data-testid="no-results"
      className="flex flex-col items-center justify-center rounded-card border border-edge bg-surface px-6 py-16 text-center"
    >
      <span aria-hidden="true" className="mb-4 text-3xl text-ink-faint/60">
        ⌕
      </span>
      <h2 className="text-base font-medium text-ink">{describe(filter)}</h2>
      <p className="mt-2 max-w-sm text-sm text-ink-muted">
        Your vault holds {total} {total === 1 ? 'collectible' : 'collectibles'}, but none matching
        this.{' '}
        <Link href="/collection" className="text-ink underline-offset-4 hover:underline">
          See everything
        </Link>
        .
      </p>
    </div>
  )
}
