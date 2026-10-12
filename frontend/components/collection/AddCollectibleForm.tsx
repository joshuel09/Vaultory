'use client'

import { useRef, useState } from 'react'
import { CollectibleForm, optional, type CollectibleFormValues } from './CollectibleForm'
import { addCollectible } from '@/lib/api/collectibles'
import { randomUUID } from '@/lib/uuid'
import type { CollectibleImageRef, CollectionStatus } from '@/lib/api/types'

/**
 * Record a new collectible.
 *
 * The form itself is shared with editing; what belongs to adding alone is the submission key and
 * what happens after a successful save.
 */
export function AddCollectibleForm() {
  const [saved, setSaved] = useState<string | null>(null)

  /*
   * One submission key per collectible, not per attempt (FR-047 of feature 001).
   *
   * This is the whole point: a double-click or a retry after a lost response replays the same key
   * and returns the collectible already created, while a collector deliberately adding a second
   * identical copy starts a new form and so a new key. Regenerating per attempt would defeat it.
   */
  const submissionKey = useRef<string>(randomUUID())

  async function handleSubmit(values: CollectibleFormValues, image: CollectibleImageRef | null) {
    const created = await addCollectible({
      submissionKey: submissionKey.current,
      name: values.name,
      collectionStatus: values.collectionStatus as CollectionStatus,
      character: optional(values.character),
      series: optional(values.series),
      manufacturer: optional(values.manufacturer),
      category: optional(values.category),
      scale: optional(values.scale),
      edition: optional(values.edition),
      purchasePrice: optional(values.purchasePrice),
      purchaseDate: optional(values.purchaseDate),
      releaseDate: optional(values.releaseDate),
      notes: optional(values.notes),
      imageId: image?.id,
    })

    setSaved(created.name)
    // A fresh key: the next collectible is a new submission, not a retry of this one.
    submissionKey.current = randomUUID()

    // Deliberately no router.refresh() here.
    //
    // It refreshed the route the collector is still on — this form — which is static, so it
    // changed nothing. The collection page is force-dynamic and re-renders on navigation
    // regardless, so the data is fresh when they get there either way.
    //
    // It was not harmless: the refresh is a navigation to /collection/new, and leaving the page
    // while it was in flight aborted the next one. That surfaced as "navigation interrupted by
    // another navigation", attributed to whichever page was being opened rather than to the add
    // that had not finished settling.
  }

  return (
    <CollectibleForm
      submitLabel="Add to my vault"
      submittingLabel="Adding…"
      hint="A name and a status are all you need."
      resetOnSuccess
      onSubmit={handleSubmit}
      notice={
        saved && (
          <div
            data-testid="add-success"
            role="status"
            className="rounded-lg border border-success/40 bg-success/10 px-4 py-3 text-sm text-ink"
          >
            <span aria-hidden="true" className="mr-2 text-success">
              ✓
            </span>
            <strong className="font-medium">{saved}</strong> was added to your vault.
          </div>
        )
      }
    />
  )
}
