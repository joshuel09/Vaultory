'use client'

import { useRouter } from 'next/navigation'
import { CollectibleForm, optional, valuesOf, type CollectibleFormValues } from './CollectibleForm'
import { editCollectible } from '@/lib/api/collectibles'
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

  async function handleSubmit(values: CollectibleFormValues, image: CollectibleImageRef | null) {
    await editCollectible(collectible.id, {
      expectedVersion: collectible.version,
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

  return (
    <CollectibleForm
      initialValues={valuesOf(collectible)}
      initialImage={collectible.image ?? null}
      submitLabel="Save changes"
      submittingLabel="Saving…"
      onSubmit={handleSubmit}
    />
  )
}
