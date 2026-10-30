'use client'

import Link from 'next/link'
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { ApiError } from '@/lib/api/errors'
import { listTrash, restoreCollectible } from '@/lib/api/trash'
import type { TrashedCollectible } from '@/lib/api/types'
import { DeletePermanentlyDialog } from './DeletePermanentlyDialog'
import { ImagePlaceholder } from './ImagePlaceholder'

interface Props {
  initialItems: TrashedCollectible[]
  initialCursor: string | null
}

/**
 * The collector's Trash: what they deleted in the last 30 days, most recent first, each with its
 * photograph and the day it will be gone for good (feature 010, FR-006).
 *
 * The date comes from the server's purgeAt. Retention is a rule of the system, so the browser
 * shows it and never works it out (Principle II).
 */
export function TrashList({ initialItems, initialCursor }: Props) {
  const [items, setItems] = useState(initialItems)
  const [cursor, setCursor] = useState(initialCursor)
  const [loading, setLoading] = useState(false)
  const [restoring, setRestoring] = useState<string | null>(null)
  const [message, setMessage] = useState<{ text: string; restoredId?: string } | null>(null)

  function remove(id: string) {
    setItems((prev) => prev.filter((it) => it.collectible.id !== id))
  }

  async function restore(item: TrashedCollectible) {
    if (restoring) return
    setRestoring(item.collectible.id)
    setMessage(null)
    try {
      await restoreCollectible(item.collectible.id)
      remove(item.collectible.id)
      setMessage({ text: `${item.collectible.name} is back in your vault.`, restoredId: item.collectible.id })
    } catch (err) {
      if (err instanceof ApiError && err.isNotFound) {
        // Already restored elsewhere, or gone for good: either way it no longer belongs here.
        remove(item.collectible.id)
        setMessage({ text: `${item.collectible.name} could not be restored — it is no longer in the Trash.` })
      } else {
        setMessage({
          text:
            err instanceof ApiError && err.isUnauthenticated
              ? 'Your session has expired. Sign in and try again.'
              : `${item.collectible.name} could not be restored just now. Try again.`,
        })
      }
    } finally {
      setRestoring(null)
    }
  }

  async function loadMore() {
    if (!cursor || loading) return
    setLoading(true)
    try {
      const page = await listTrash({ cursor })
      setItems((prev) => [...prev, ...page.items])
      setCursor(page.nextCursor ?? null)
    } catch {
      setMessage({ text: 'More of the Trash could not be loaded. Try again.' })
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-6">
      {message && (
        <p role="status" data-testid="trash-message" className="rounded-lg border border-edge bg-surface-raised px-4 py-3 text-sm text-ink">
          {message.text}{' '}
          {message.restoredId && (
            <Link href="/collection" className="underline underline-offset-4 hover:text-accent">
              View your vault
            </Link>
          )}
        </p>
      )}

      {items.length === 0 ? (
        <div
          data-testid="empty-trash"
          className="flex flex-col items-center justify-center rounded-card border border-dashed border-edge px-6 py-20 text-center"
        >
          <h2 className="text-lg font-medium text-ink">The Trash is empty</h2>
          <p className="mt-2 max-w-sm text-sm text-ink-muted">
            When you delete a collectible it comes here, with its photograph, and is kept for 30 days
            in case you want it back.
          </p>
        </div>
      ) : (
        <ul className="divide-y divide-edge rounded-card border border-edge bg-surface">
          {items.map((item) => {
            const { collectible, purgeAt } = item
            return (
              <li key={collectible.id} data-testid="trashed-collectible" className="flex items-center gap-4 p-3.5">
                <div className="aspect-collectible w-16 shrink-0 overflow-hidden rounded-lg bg-surface-raised">
                  {collectible.image ? (
                    // A plain <img> for the same reason as the gallery card: the rendition is
                    // served from an authorized path the optimizer cannot reach.
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={collectible.image.renditionUrl}
                      alt={`Photograph of ${collectible.name}`}
                      loading="lazy"
                      className="h-full w-full object-cover"
                    />
                  ) : (
                    <ImagePlaceholder name={collectible.name} />
                  )}
                </div>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium text-ink" title={collectible.name}>
                    {collectible.name}
                  </p>
                  <p className="text-xs text-ink-faint">
                    Removed for good on <time dateTime={purgeAt}>{formatDate(purgeAt)}</time>
                  </p>
                </div>
                <div className="flex shrink-0 flex-wrap items-center justify-end gap-2">
                  <Button
                    type="button"
                    size="sm"
                    variant="secondary"
                    data-testid="restore-collectible"
                    aria-label={`Restore ${collectible.name}`}
                    disabled={restoring !== null}
                    onClick={() => restore(item)}
                  >
                    {restoring === collectible.id ? 'Restoring…' : 'Restore'}
                  </Button>
                  <DeletePermanentlyDialog
                    collectibleId={collectible.id}
                    name={collectible.name}
                    onDeleted={() => {
                      remove(collectible.id)
                      setMessage({ text: `${collectible.name} was deleted permanently.` })
                    }}
                  />
                </div>
              </li>
            )
          })}
        </ul>
      )}

      {cursor && (
        <div className="flex justify-center">
          <Button type="button" variant="secondary" disabled={loading} onClick={loadMore}>
            {loading ? 'Loading…' : 'Show more'}
          </Button>
        </div>
      )}
    </div>
  )
}

/** A date the collector reads, in their own locale. */
function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'long', year: 'numeric' })
}
