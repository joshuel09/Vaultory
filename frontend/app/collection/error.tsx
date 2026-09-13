'use client'

import { useEffect } from 'react'
import { Button } from '@/components/ui/button'

/**
 * Shown when the collection cannot be retrieved (FR-043).
 *
 * It says what happened and offers a way to try again. It does not show an empty gallery, which
 * would tell a collector their vault is empty when it is merely unreachable.
 */
export default function CollectionError({
  error,
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}) {
  useEffect(() => {
    // Recorded for diagnosis; the collector is shown none of it.
    console.error('collection view failed', error)
  }, [error])

  return (
    <main className="mx-auto w-full max-w-2xl px-4 py-24">
      <div
        data-testid="collection-error"
        role="alert"
        className="flex flex-col items-center rounded-card border border-danger/40 bg-surface px-6 py-14 text-center"
      >
        <span aria-hidden="true" className="mb-4 text-3xl text-danger">
          !
        </span>
        <h1 className="text-lg font-medium text-ink">Your collection could not be loaded</h1>
        <p className="mt-2 max-w-sm text-sm text-ink-muted">
          Nothing in your vault has changed. This is a problem reaching it, not with what is in it.
        </p>
        <Button className="mt-6" onClick={reset}>
          Try again
        </Button>
      </div>
    </main>
  )
}
