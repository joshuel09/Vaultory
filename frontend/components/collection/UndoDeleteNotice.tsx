'use client'

import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { ApiError } from '@/lib/api/errors'
import { restoreCollectible } from '@/lib/api/trash'

interface Props {
  /** The collectible just moved to the Trash. */
  deletedId: string
  /** The gallery as the collector was looking at it, without the deletion marker. */
  galleryHref: string
}

/**
 * "Moved to the Trash — Undo", on the gallery straight after a deletion (feature 010, FR-005).
 *
 * Most mistaken deletions are noticed at once, so the way back is offered where the collector
 * lands, not three screens away. Undo is the same restore the Trash offers (FR-007): one mechanism,
 * one set of guarantees.
 *
 * Shown once. The marker that summons it is in the URL, and every way out of the notice — undoing,
 * dismissing — replaces the URL without it, so a refresh or a back-navigation does not offer to undo
 * something already dealt with.
 */
export function UndoDeleteNotice({ deletedId, galleryHref }: Props) {
  const router = useRouter()
  const [restoring, setRestoring] = useState(false)
  const [problem, setProblem] = useState<string | null>(null)

  function leave() {
    router.replace(galleryHref, { scroll: false })
    router.refresh()
  }

  async function undo() {
    if (restoring) return
    setRestoring(true)
    setProblem(null)
    try {
      await restoreCollectible(deletedId)
      leave()
    } catch (err) {
      setRestoring(false)
      setProblem(
        err instanceof ApiError && err.isNotFound
          ? 'It could not be brought back — it may already have been restored or deleted for good.'
          : err instanceof ApiError && err.isUnauthenticated
            ? 'Your session has expired. Sign in and try again.'
            : 'It could not be brought back just now. It is still in the Trash — try again.',
      )
    }
  }

  return (
    <div
      role="status"
      data-testid="undo-delete-notice"
      className="mb-6 flex flex-wrap items-center justify-between gap-3 rounded-lg border border-edge bg-surface-raised px-4 py-3 text-sm text-ink"
    >
      <p>
        Moved to the{' '}
        <Link href="/collection/trash" className="underline underline-offset-4 hover:text-accent">
          Trash
        </Link>
        . It will be kept for 30 days.
        {problem && (
          <span role="alert" className="mt-1 block text-danger">
            {problem}
          </span>
        )}
      </p>
      <div className="flex items-center gap-2">
        <Button type="button" size="sm" variant="secondary" disabled={restoring} onClick={undo}>
          {restoring ? 'Restoring…' : 'Undo'}
        </Button>
        <Button type="button" size="sm" variant="ghost" aria-label="Dismiss" onClick={leave}>
          ✕
        </Button>
      </div>
    </div>
  )
}
