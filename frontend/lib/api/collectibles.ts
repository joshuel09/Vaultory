import { toApiError } from './errors'
import type {
  AddCollectibleRequest,
  Collectible,
  CollectionPage,
  CollectionStatus,
  EditCollectibleRequest,
} from './types'

/**
 * Requests go to /api/* on this origin, which Next.js rewrites to the Go service. Same-origin is
 * what carries the session cookie, including on the <img> requests that fetch renditions
 * (research.md Decision 2).
 */
const API = '/api'

export interface ListParams {
  status?: CollectionStatus
  cursor?: string
  limit?: number
}

export async function listCollectibles(
  params: ListParams = {},
  init?: RequestInit,
): Promise<CollectionPage> {
  const query = new URLSearchParams()
  if (params.status) query.set('status', params.status)
  if (params.cursor) query.set('cursor', params.cursor)
  if (params.limit) query.set('limit', String(params.limit))

  const suffix = query.size > 0 ? `?${query.toString()}` : ''
  const response = await fetch(`${API}/collectibles${suffix}`, {
    ...init,
    // A collection is per-collector and changes as they add to it; never serve a cached copy.
    cache: 'no-store',
    credentials: 'same-origin',
  })
  if (!response.ok) throw await toApiError(response)
  return (await response.json()) as CollectionPage
}

export async function addCollectible(body: AddCollectibleRequest): Promise<Collectible> {
  const response = await fetch(`${API}/collectibles`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'same-origin',
    body: JSON.stringify(body),
  })
  if (!response.ok) throw await toApiError(response)
  return (await response.json()) as Collectible
}

/**
 * Read one collectible, which is what the edit screen is filled from.
 *
 * A collectible belonging to another collector answers 404, exactly as one that never existed, so
 * nothing here needs to — or can — tell the two apart (FR-031).
 */
export async function getCollectible(id: string, init?: RequestInit): Promise<Collectible> {
  const response = await fetch(`${API}/collectibles/${encodeURIComponent(id)}`, {
    ...init,
    cache: 'no-store',
    credentials: 'same-origin',
  })
  if (!response.ok) throw await toApiError(response)
  return (await response.json()) as Collectible
}

/**
 * Replace every attribute of one collectible.
 *
 * A full replacement, not a patch: whatever is not sent is cleared, and `imageId` in particular
 * removes the photograph when it is null (FR-017, FR-018).
 *
 * A stale `expectedVersion` comes back as 409 rather than overwriting the newer values (FR-027);
 * `toApiError` carries that through so the caller can show what the collectible now says.
 */
export async function editCollectible(
  id: string,
  body: EditCollectibleRequest,
): Promise<Collectible> {
  const response = await fetch(`${API}/collectibles/${encodeURIComponent(id)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'same-origin',
    body: JSON.stringify(body),
  })
  if (!response.ok) throw await toApiError(response)
  return (await response.json()) as Collectible
}

/**
 * Delete a collectible permanently.
 *
 * Succeeds whether or not anything was there: already deleted, never existed, or another
 * collector's all answer the same way (FR-025). A retry after a lost response is therefore safe,
 * and no response distinguishes a real identifier from a fictional one (FR-031).
 */
export async function deleteCollectible(id: string): Promise<void> {
  const response = await fetch(`${API}/collectibles/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    credentials: 'same-origin',
  })
  if (!response.ok) throw await toApiError(response)
}
