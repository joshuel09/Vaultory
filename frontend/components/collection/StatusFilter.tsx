'use client'

import { usePathname, useRouter, useSearchParams } from 'next/navigation'
import { cn } from '@/lib/cn'
import { COLLECTION_STATUSES, STATUS_LABELS, type CollectionStatus } from '@/lib/api/types'

/**
 * Narrow the gallery to one status, or return to all of it (FR-037, FR-038, FR-039).
 *
 * The active filter lives in the URL, so a filtered view survives a reload and can be shared. A
 * radio group rather than buttons, because the choice is single-select and radios give keyboard
 * arrow navigation and the right semantics for free (FR-045).
 */
export function StatusFilter({ active }: { active: CollectionStatus | null }) {
  const router = useRouter()
  const pathname = usePathname()
  const params = useSearchParams()

  function select(status: CollectionStatus | null) {
    const next = new URLSearchParams(params.toString())
    if (status) {
      next.set('status', status)
    } else {
      next.delete('status')
    }
    // Changing the filter restarts paging; a cursor from the old filter means nothing under the
    // new one.
    next.delete('cursor')
    const query = next.toString()
    router.push(query ? `${pathname}?${query}` : pathname)
  }

  const options: Array<{ value: CollectionStatus | null; label: string }> = [
    { value: null, label: 'All' },
    ...COLLECTION_STATUSES.map((s) => ({ value: s, label: STATUS_LABELS[s] })),
  ]

  return (
    <fieldset data-testid="status-filter">
      <legend className="sr-only">Filter your collection by status</legend>
      <div role="radiogroup" aria-label="Filter by status" className="flex flex-wrap gap-2">
        {options.map(({ value, label }) => {
          const isActive = active === value
          return (
            <label
              key={label}
              className={cn(
                'cursor-pointer select-none rounded-full border px-3.5 py-1.5 text-sm transition-colors',
                isActive
                  // The active filter is marked by weight, border, and fill — not colour alone.
                  ? 'border-accent bg-accent/15 font-medium text-accent'
                  : 'border-edge text-ink-muted hover:border-ink-faint hover:text-ink',
              )}
            >
              <input
                type="radio"
                name="status"
                className="sr-only"
                checked={isActive}
                onChange={() => select(value)}
              />
              {label}
              {isActive && <span className="sr-only"> (active filter)</span>}
            </label>
          )
        })}
      </div>
    </fieldset>
  )
}
