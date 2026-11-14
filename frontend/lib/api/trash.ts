import { toApiError } from './errors'
import type { Collectible, TrashPage } from './types'

/**
 * The Trash (feature 010). Same-origin, through the rewrite, like every other browser request.
 */
const API = '/api/trash'

export async function listTrash(
  params: { cursor?: string; limit?: number } = {},
  init?: RequestInit,
): Promise<TrashPage> {
  const query = new URLSearchParams()
  if (params.cursor) query.set('cursor', params.cursor)
  if (params.limit) query.set('limit', String(params.limit))
  const suffix = query.size > 0 ? `?${query.toString()}` : ''
  const response = await fetch(`${API}${suffix}`, {
    ...init,
    cache: 'no-store',
    credentials: 'same-origin',
  })
  if (!response.ok) throw await toApiError(response)
  return (await response.json()) as TrashPage
}

/**
 * Return a collectible from the Trash to the vault, exactly as it was.
 *
 * Throws a 404 ApiError when there is nothing to restore — already restored, gone for good, or past
 * its 30 days. Unlike deleting, that is worth telling the collector: it did not come back (FR-008).
 */
export async function restoreCollectible(id: string): Promise<Collectible> {
  const response = await fetch(`${API}/${encodeURIComponent(id)}/restore`, {
    method: 'POST',
    credentials: 'same-origin',
  })
  if (!response.ok) throw await toApiError(response)
  return (await response.json()) as Collectible
}

/** Destroy a collectible in the Trash, and its photograph, now. Succeeds whatever was there. */
export async function deletePermanently(id: string): Promise<void> {
  const response = await fetch(`${API}/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    credentials: 'same-origin',
  })
  if (!response.ok) throw await toApiError(response)
}
