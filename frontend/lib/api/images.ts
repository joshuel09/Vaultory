import { toApiError } from './errors'
import type { CollectibleImage } from './types'

/** What the server accepts. Mirrored here only to fail fast; the server's verdict is the one that
 *  counts, and these values are repeated in its refusal messages (FR-009, FR-010). */
export const ACCEPTED_IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/webp'] as const
export const MAX_IMAGE_BYTES = 10 * 1024 * 1024

export async function uploadImage(file: File): Promise<CollectibleImage> {
  const body = new FormData()
  body.append('file', file)

  const response = await fetch('/api/images', {
    method: 'POST',
    credentials: 'same-origin',
    body,
  })
  if (!response.ok) throw await toApiError(response)
  return (await response.json()) as CollectibleImage
}

/**
 * A courtesy check so an obviously wrong file fails instantly instead of after a 10 MB upload.
 *
 * It is not load-bearing: the server re-checks by decoding the bytes, because a browser's reported
 * type is just a claim (FR-009, Constitution III).
 */
export function describeObviousProblem(file: File): string | null {
  if (file.size > MAX_IMAGE_BYTES) {
    return 'That image is larger than the 10 MB limit.'
  }
  if (file.type && !(ACCEPTED_IMAGE_TYPES as readonly string[]).includes(file.type)) {
    return 'Vaultory accepts JPEG, PNG, and WebP images.'
  }
  return null
}
