import Link from 'next/link'
import type { GroupSummary } from '@/lib/api/types'
import { hrefForGroup, type GroupDimension } from '@/lib/api/statistics-links'
import { formatMoney } from '@/lib/format-money'

/**
 * One breakdown — by series, or by manufacturer — shared by both dimensions.
 *
 * The order is the backend's. Named groups come first, largest first, with the unspecified group
 * and the remainder below them however large they are, because "everything else" standing above a
 * real series reads as though the collection is mostly nothing in particular (FR-022). Reordering
 * here would be the presentation layer overruling that.
 */

interface Props {
  dimension: GroupDimension
  groups: GroupSummary[]
  heading: string
  /** What to say when the backend returned no groups at all for this dimension. */
  emptyNote: string
}

export function GroupBreakdown({ dimension, groups, heading, emptyNote }: Props) {
  if (groups.length === 0) {
    return (
      <section data-testid={`breakdown-${dimension}`} aria-labelledby={`breakdown-${dimension}-heading`}>
        <h2 id={`breakdown-${dimension}-heading`} className="text-lg font-medium tracking-tight text-ink">
          {heading}
        </h2>
        <p className="mt-2 text-sm text-ink-muted">{emptyNote}</p>
      </section>
    )
  }

  // The largest group, for the bar widths. A proportion of something already on screen, not a
  // figure about the vault — nothing here is reported to the collector as a number.
  const widest = groups.reduce((most, g) => Math.max(most, g.count), 0) || 1

  return (
    <section data-testid={`breakdown-${dimension}`} aria-labelledby={`breakdown-${dimension}-heading`}>
      <h2 id={`breakdown-${dimension}-heading`} className="text-lg font-medium tracking-tight text-ink">
        {heading}
      </h2>

      <ul className="mt-4 space-y-px overflow-hidden rounded-xl border border-edge">
        {groups.map((group, index) => {
          const href = hrefForGroup(dimension, group)
          const body = (
            <>
              <div className="flex items-baseline justify-between gap-4">
                <span
                  data-testid={`group-${dimension}-${index}-label`}
                  className="truncate text-sm font-medium text-ink"
                >
                  {group.label}
                </span>
                <span className="shrink-0 text-sm text-ink-muted">
                  <span data-testid={`group-${dimension}-${index}-count`} className="text-ink">
                    {group.count}
                  </span>{' '}
                  {group.count === 1 ? 'piece' : 'pieces'} ·{' '}
                  <span data-testid={`group-${dimension}-${index}-total`}>
                    {formatMoney(group.total)}
                  </span>
                </span>
              </div>
              {/* Decorative: the figures above are the information, and this only makes their
                  relative size readable at a glance. */}
              <div aria-hidden="true" className="mt-2 h-1 rounded-full bg-surface">
                <div
                  className="h-1 rounded-full bg-accent/60"
                  style={{ width: `${Math.max(2, Math.round((group.count / widest) * 100))}%` }}
                />
              </div>
            </>
          )

          return (
            <li key={`${group.kind}-${group.label}-${index}`} className="bg-surface-raised">
              {href === null ? (
                /*
                  The remainder, which no filter reproduces: it means everything past the groups
                  shown, and the gallery cannot be asked that. A link that returned something else
                  would be worse than none, so it is text, and the note says why it cannot be opened
                  rather than leaving a collector clicking at it.
                */
                <div data-testid={`group-${dimension}-${index}`} className="px-4 py-3">
                  {body}
                  <p className="mt-2 text-xs text-ink-faint">
                    The groups beyond those shown, counted together.
                  </p>
                </div>
              ) : (
                <Link
                  href={href}
                  data-testid={`group-${dimension}-${index}`}
                  className="block px-4 py-3 transition-colors hover:bg-surface focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-accent"
                >
                  {body}
                </Link>
              )}
            </li>
          )
        })}
      </ul>
    </section>
  )
}
