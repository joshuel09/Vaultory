'use client'

import { useEffect, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { ApiError } from '@/lib/api/errors'
import { deleteCollectible } from '@/lib/api/collectibles'

interface Props {
  collectibleId: string
  /** Named in the confirmation, so the collector can see which one they are about to lose. */
  name: string
  /** Where to send them once it is gone. */
  onDeleted: () => void
}

/**
 * Delete a collectible, behind a confirmation proportionate to the fact that it cannot be undone
 * (FR-023).
 *
 * A native <dialog> opened with showModal(), which gives focus containment, Escape to dismiss, the
 * inert backdrop and focus restoration on close for nothing. A dialog library would be a
 * dependency carried to reimplement what the element already does, and Principle V puts the burden
 * of justification on the dependency (research.md Decision 9).
 *
 * What makes the confirmation proportionate rather than merely present:
 *
 *   - it names the collectible, so there is no doubt which one is about to go;
 *   - it says plainly that this cannot be undone, because it cannot;
 *   - Cancel holds the initial focus and the dialog's default action is to cancel, so pressing
 *     Enter on an untouched dialog deletes nothing;
 *   - it does not demand the collector type the name. That is the friction appropriate to bulk or
 *     account-level destruction; for one entry it is theatre.
 */
export function DeleteCollectibleDialog({ collectibleId, name, onDeleted }: Props) {
  const dialog = useRef<HTMLDialogElement | null>(null)
  const cancel = useRef<HTMLButtonElement | null>(null)
  const [open, setOpen] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const el = dialog.current
    if (!el) return
    if (open && !el.open) {
      el.showModal()
      // Focus the way out, not the way through. The destructive button is never what a stray
      // keypress lands on.
      cancel.current?.focus()
    } else if (!open && el.open) {
      el.close()
    }
  }, [open])

  async function handleDelete() {
    // One deletion at a time (FR-036). Harmless on the server, which answers a repeat with 204
    // anyway, but a second request would also fire a second navigation.
    if (deleting) return
    setDeleting(true)
    setError(null)
    try {
      await deleteCollectible(collectibleId)
      setOpen(false)
      onDeleted()
    } catch (err) {
      setError(
        err instanceof ApiError && err.isUnauthenticated
          ? 'Your session has expired. Sign in and try again.'
          : 'This could not be deleted. Nothing has changed — try again.',
      )
      setDeleting(false)
    }
  }

  return (
    <>
      <Button
        type="button"
        variant="ghost"
        data-testid="delete-collectible"
        onClick={() => {
          setError(null)
          setOpen(true)
        }}
        className="text-danger hover:bg-danger/10 hover:text-danger"
      >
        Delete this collectible
      </Button>

      <dialog
        ref={dialog}
        data-testid="delete-dialog"
        aria-labelledby="delete-dialog-title"
        // Escape, the backdrop, and the close button all mean the same thing: do not delete.
        onClose={() => {
          setOpen(false)
          setDeleting(false)
        }}
        className="max-w-md rounded-card border border-edge bg-surface p-0 text-ink backdrop:bg-black/60 open:animate-none"
      >
        <form method="dialog" className="space-y-5 p-6">
          <h2 id="delete-dialog-title" className="text-lg font-semibold tracking-tight">
            Delete {name}?
          </h2>
          <p className="text-sm text-ink-muted">
            This cannot be undone. {name} and its photograph will be removed from your vault for
            good — there is no trash to recover it from.
          </p>

          {error && (
            <p role="alert" className="rounded-lg border border-danger/40 bg-danger/10 px-3 py-2 text-sm">
              {error}
            </p>
          )}

          <div className="flex flex-wrap items-center justify-end gap-3">
            {/*
              * Cancel is the dialog's default: it is focused on open, and because the form's method
              * is "dialog" with no value, Enter on an untouched dialog closes without deleting
              * (FR-023).
              */}
            <Button type="submit" variant="secondary" ref={cancel} data-testid="cancel-delete">
              Keep it
            </Button>
            <Button
              type="button"
              data-testid="confirm-delete"
              disabled={deleting}
              onClick={handleDelete}
              className="bg-danger text-white hover:bg-danger/90"
            >
              {deleting ? 'Deleting…' : 'Delete permanently'}
            </Button>
          </div>
        </form>
      </dialog>
    </>
  )
}
