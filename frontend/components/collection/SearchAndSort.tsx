'use client'

import { usePathname, useRouter, useSearchParams } from 'next/navigation'
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { DEFAULT_SORT, SORT_LABELS, type SortOrder } from '@/lib/api/types'

interface Props {
  /** The search the gallery is showing, so the field starts with it. */
  q: string | null
  sort: SortOrder | null
}

/**
 * Search the vault and choose its order (feature 011, FR-001, FR-006).
 *
 * Both live in the address, beside the status and series filters, so a view survives a reload, can
 * be shared, and comes back after editing (FR-011). Changing either keeps every parameter it does not
 * own — and drops the page reference, which means nothing under a new search or order (FR-012).
 *
 * Native controls, deliberately: a labelled search field submitted with Enter, and a select that
 * applies on change. Keyboard operation and assistive-technology labels come with the elements
 * (FR-015).
 */
export function SearchAndSort({ q, sort }: Props) {
  const router = useRouter()
  const pathname = usePathname()
  const params = useSearchParams()
  const [text, setText] = useState(q ?? '')

  function go(change: (next: URLSearchParams) => void) {
    const next = new URLSearchParams(params.toString())
    change(next)
    next.delete('cursor')
    // A fresh view is not "just deleted" any more; the Undo notice belongs to the moment after.
    next.delete('deleted')
    const query = next.toString()
    router.push(query ? `${pathname}?${query}` : pathname)
  }

  function applySearch(value: string) {
    go((next) => {
      const trimmed = value.trim()
      if (trimmed) next.set('q', trimmed)
      else next.delete('q')
    })
  }

  return (
    <div data-testid="search-and-sort" className="flex flex-wrap items-end gap-3">
      <form
        role="search"
        className="flex min-w-0 flex-1 basis-64 items-end gap-2"
        onSubmit={(e) => {
          e.preventDefault()
          applySearch(text)
        }}
      >
        <label className="min-w-0 flex-1">
          <span className="mb-1 block text-xs font-medium text-ink-muted">Search your vault</span>
          <input
            type="search"
            name="q"
            value={text}
            maxLength={200}
            placeholder="Name, character, series, maker…"
            onChange={(e) => setText(e.target.value)}
            className="w-full rounded-lg border border-edge bg-surface px-3 py-2 text-sm text-ink placeholder:text-ink-faint focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
          />
        </label>
        <Button type="submit" variant="secondary" size="sm">
          Search
        </Button>
        {q && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => {
              setText('')
              applySearch('')
            }}
          >
            Clear
          </Button>
        )}
      </form>

      <label>
        <span className="mb-1 block text-xs font-medium text-ink-muted">Sort by</span>
        <select
          name="sort"
          value={sort ?? DEFAULT_SORT}
          onChange={(e) =>
            go((next) => {
              const value = e.target.value as SortOrder
              if (value === DEFAULT_SORT) next.delete('sort')
              else next.set('sort', value)
            })
          }
          className="rounded-lg border border-edge bg-surface px-3 py-2 text-sm text-ink focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
        >
          {(Object.keys(SORT_LABELS) as SortOrder[]).map((o) => (
            <option key={o} value={o}>
              {SORT_LABELS[o]}
            </option>
          ))}
        </select>
      </label>
    </div>
  )
}
