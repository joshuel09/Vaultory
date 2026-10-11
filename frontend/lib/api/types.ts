import type { components } from '@/lib/types/api'

/**
 * Named aliases over the generated contract types.
 *
 * Everything here derives from contracts/openapi.yaml. Nothing in this application declares a
 * request or response shape by hand, so the contract stays the single source of truth
 * (Constitution Principle III).
 */
export type Collectible = components['schemas']['Collectible']
export type CollectionPage = components['schemas']['CollectionPage']
export type CollectionStatus = components['schemas']['CollectionStatus']
export type AddCollectibleRequest = components['schemas']['AddCollectibleRequest']
export type EditCollectibleRequest = components['schemas']['EditCollectibleRequest']
export type VersionConflictResponse = components['schemas']['VersionConflictResponse']
export type CollectibleImage = components['schemas']['CollectibleImage']
/**
 * What a collectible carries: enough to render its photograph, and nothing about the upload.
 *
 * Distinct from CollectibleImage, which is the upload response and also reports the content type
 * and byte size. An uploaded image satisfies this shape, which is why the picker can hand one
 * straight to the form — but the two are not interchangeable in the other direction.
 */
export type CollectibleImageRef = components['schemas']['CollectibleImageRef']
export type ErrorResponse = components['schemas']['ErrorResponse']
export type FieldError = NonNullable<ErrorResponse['error']['fields']>[number]

/** The four statuses, in the order a collector reads them. */
export const COLLECTION_STATUSES = ['owned', 'preordered', 'wishlist', 'sold'] as const

export const STATUS_LABELS: Record<CollectionStatus, string> = {
  owned: 'Owned',
  preordered: 'Preordered',
  wishlist: 'Wishlist',
  sold: 'Sold',
}

export function isCollectionStatus(value: string): value is CollectionStatus {
  return (COLLECTION_STATUSES as readonly string[]).includes(value)
}
