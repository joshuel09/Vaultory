'use client'

import { useEffect, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { ApiError } from '@/lib/api/errors'
import { deletePermanently } from '@/lib/api/trash'

interface Props {
  collectibleId: string
  name: string
  onDeleted: () => void
}

/**
 * Destroy a collectible in the Trash, now (feature 010, FR-011).
 *
 * The confirmation feature 006 built for permanent deletion, moved here — because this is where
 * deletion is now permanent. The same properties make it proportionate: it names the collectible,
 * it says plainly that this cannot be undone, Cancel holds focus, and Enter on an untouched dialog
 * cancels (006 FR-023).
 */
export function DeletePermanentlyDialog({ collectibleId, name, onDeleted }: Props) {
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
      cancel.current?.focus()
    } else if (!open && el.open) {
      el.close()
    }
  }, [open])

  async function handleDelete() {
    if (deleting) return
    setDeleting(true)
    setError(null)
    try {
      await deletePermanently(collectibleId)
      setOpen(false)
      onDeleted()
    } catch (err) {
      setError(
        err instanceof ApiError && err.isUnauthenticated
          ? 'Your session has expired. Sign in and try again.'
          : 'This could not be deleted. It is still in the Trash — try again.',
      )
      setDeleting(false)
    }
  }

  return (
    <>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        data-testid="delete-permanently"
        aria-label={`Delete ${name} permanently`}
        onClick={() => {
          setError(null)
          setOpen(true)
        }}
        className="text-danger hover:bg-danger/10 hover:text-danger"
      >
        Delete permanently
      </Button>

      <dialog
        ref={dialog}
        data-testid="delete-permanently-dialog"
        aria-labelledby={`delete-permanently-title-${collectibleId}`}
        onClose={() => {
          setOpen(false)
          setDeleting(false)
        }}
        className="max-w-md rounded-card border border-edge bg-surface p-0 text-ink backdrop:bg-black/60"
      >
        <form method="dialog" className="space-y-5 p-6">
          <h2 id={`delete-permanently-title-${collectibleId}`} className="text-lg font-semibold tracking-tight">
            Delete {name} permanently?
          </h2>
          <p className="text-sm text-ink-muted">
            This cannot be undone. {name} and its photograph will be gone for good.
          </p>
          {error && (
            <p role="alert" className="rounded-lg border border-danger/40 bg-danger/10 px-3 py-2 text-sm">
              {error}
            </p>
          )}
          <div className="flex flex-wrap items-center justify-end gap-3">
            {/* The dialog's default is to keep it: focused on open, and Enter on an untouched dialog
                submits this method="dialog" form, which closes without deleting. */}
            <Button type="submit" variant="secondary" ref={cancel} data-testid="cancel-delete-permanently">
              Keep it
            </Button>
            <Button
              type="button"
              data-testid="confirm-delete-permanently"
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
