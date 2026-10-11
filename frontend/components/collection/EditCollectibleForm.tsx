'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { CollectibleForm, optional, valuesOf, type CollectibleFormValues } from './CollectibleForm'
import { VersionConflictNotice } from './VersionConflictNotice'
import { DeleteCollectibleDialog } from './DeleteCollectibleDialog'
import { editCollectible } from '@/lib/api/collectibles'
import { ApiError } from '@/lib/api/errors'
import type { Collectible, CollectibleImageRef, CollectionStatus } from '@/lib/api/types'

interface Props {
  collectible: Collectible
  /** Where to go after a successful save. Already passed through safeRedirect by the page. */
  returnTo: string
}

/**
 * Change a collectible that is already in the vault.
 *
 * The form is the one adding uses, so the two cannot disagree about what is valid (SC-008). What
 * belongs to editing alone is the version it was opened at, and the rule about the photograph
 * below.
 */
export function EditCollectibleForm({ collectible, returnTo }: Props) {
  const router = useRouter()

  /*
   * The collectible the form is currently based on, and the version its save will present.
   *
   * It moves only when the collector asks it to, by accepting the values a conflict reported.
   * Changing it for them would be the silent overwrite this mechanism exists to prevent, in the
   * other direction.
   */
  const [base, setBase] = useState<Collectible>(collectible)
  const [conflict, setConflict] = useState<Collectible | null>(null)

  async function handleSubmit(values: CollectibleFormValues, image: CollectibleImageRef | null) {
    await editCollectible(base.id, {
      expectedVersion: base.version,
      name: values.name,
      collectionStatus: values.collectionStatus as CollectionStatus,
      character: optional(values.character) ?? null,
      series: optional(values.series) ?? null,
      manufacturer: optional(values.manufacturer) ?? null,
      category: optional(values.category) ?? null,
      scale: optional(values.scale) ?? null,
      edition: optional(values.edition) ?? null,
      purchasePrice: optional(values.purchasePrice) ?? null,
      purchaseDate: optional(values.purchaseDate) ?? null,
      releaseDate: optional(values.releaseDate) ?? null,
      notes: optional(values.notes) ?? null,
      /*
       * The sharpest edge in this feature.
       *
       * The contract is a full replacement, so a null or omitted imageId does not mean "leave the
       * photograph alone" — it means this collectible has no photograph, and removes the one it had
       * (FR-017). The form hands back whatever image it is currently holding, which is the stored
       * one unless the collector changed or removed it (FR-018).
       *
       * Get this wrong and an edit that only fixes a typo silently destroys a photograph.
       */
      imageId: image?.id ?? null,
    })

    /*
     * One navigation, and no router.refresh().
     *
     * The collection route is force-dynamic and re-renders on navigation, so a refresh would change
     * nothing — and it is a second navigation that aborts the first, which is the race that was
     * mistaken for tablet flakiness for two features.
     */
    router.push(returnTo)
  }

  /**
   * A version conflict is the parent's to present, not the form's.
   *
   * Returning true keeps the form from rendering a red error banner over something that is not an
   * error: nothing was lost, and what the collector typed is still in the fields below.
   */
  function handleError(error: unknown): boolean {
    if (error instanceof ApiError && error.isVersionConflict && error.current) {
      setConflict(error.current)
      return true
    }
    return false
  }

  return (
    <CollectibleForm
      /*
       * Remounting on a new base, rather than syncing props into state in an effect.
       *
       * Accepting the conflict's values has to replace every field at once. A key change does that
       * in one render; an effect would cascade an extra one and could show the old values briefly
       * — the same reasoning the gallery uses when the filter changes.
       */
      key={`${base.id}@${base.version}`}
      initialValues={valuesOf(base)}
      initialImage={base.image ?? null}
      submitLabel="Save changes"
      submittingLabel="Saving…"
      onSubmit={handleSubmit}
      onError={handleError}
      footer={
        /*
         * Deleting is offered here and only here (FR-022a). Never on a gallery entry: a
         * destructive action one stray click away in a browsing context is how a collection gets
         * damaged by accident.
         */
        <div className="border-t border-edge pt-6">
          <DeleteCollectibleDialog
            collectibleId={base.id}
            name={base.name}
            onDeleted={() => router.push(returnTo)}
          />
          <p className="mt-2 text-xs text-ink-faint">
            Deleting is permanent. There is no undo and no trash.
          </p>
        </div>
      }
      notice={
        conflict && (
          <VersionConflictNotice
            current={conflict}
            onUseCurrent={() => {
              setBase(conflict)
              setConflict(null)
            }}
          />
        )
      }
    />
  )
}
