'use client'

import { useCallback, useEffect, useRef, useState } from 'react'
import { CollectibleCard } from './CollectibleCard'
import { Button } from '@/components/ui/button'
import { listCollectibles } from '@/lib/api/collectibles'
import type { Collectible, CollectionStatus } from '@/lib/api/types'

interface Props {
  initialItems: Collectible[]
  initialCursor: string | null
  status: CollectionStatus | null
}

/**
 * The gallery: an image-forward grid, not a table (FR-030, FR-031).
 *
 * The first page is rendered on the server; further pages load incrementally as the collector
 * scrolls, so a large collection is never presented at once (FR-035). The grid reflows from one
 * column on a phone to six on a wide desktop, and never scrolls horizontally (FR-036).
 *
 * Changing the filter produces a different collection, not a mutation of this one. The parent
 * remounts this component with a new `key` rather than having it sync props into state in an
 * effect, which would cascade an extra render and risk showing the old filter's items briefly.
 */
export function CollectionGallery({ initialItems, initialCursor, status }: Props) {
  const [items, setItems] = useState(initialItems)
  const [cursor, setCursor] = useState(initialCursor)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const sentinel = useRef<HTMLDivElement | null>(null)

  const loadMore = useCallback(async () => {
    if (!cursor || loading) return
    setLoading(true)
    setError(null)
    try {
      const page = await listCollectibles({ cursor, status: status ?? undefined })
      setItems((prev) => [...prev, ...page.items])
      setCursor(page.nextCursor ?? null)
    } catch {
      // Failing to load more must not discard what the collector is already looking at.
      setError('More of your collection could not be loaded.')
    } finally {
      setLoading(false)
    }
  }, [cursor, loading, status])

  // Load the next page as the end of the grid comes into view, with a button as the fallback for
  // anyone who is not scrolling — keyboard users included (FR-045).
  useEffect(() => {
    const node = sentinel.current
    if (!node || !cursor) return
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) void loadMore()
      },
      { rootMargin: '400px' },
    )
    observer.observe(node)
    return () => observer.disconnect()
  }, [cursor, loadMore])

  return (
    <div>
      <ul
        data-testid="collection-gallery"
        className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6"
      >
        {items.map((collectible) => (
          <li key={collectible.id}>
            <CollectibleCard collectible={collectible} />
          </li>
        ))}
      </ul>

      <div ref={sentinel} aria-hidden="true" className="h-px" />

      {(cursor || loading || error) && (
        <div className="mt-8 flex flex-col items-center gap-3">
          {error && (
            <p role="alert" className="text-sm text-danger">
              {error}
            </p>
          )}
          {cursor && (
            <Button variant="secondary" onClick={() => void loadMore()} disabled={loading}>
              {loading ? 'Loading…' : 'Show more'}
            </Button>
          )}
          {/* Announced for screen readers, which do not benefit from the grid growing silently. */}
          <p aria-live="polite" className="sr-only">
            {loading ? 'Loading more collectibles' : `${items.length} collectibles shown`}
          </p>
        </div>
      )}
    </div>
  )
}
