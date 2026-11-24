import { toApiError } from './errors'
import type {
  AddCollectibleRequest,
  Collectible,
  CollectionPage,
  CollectionStatus,
  EditCollectibleRequest,
  SortOrder,
} from './types'

/**
 * Requests go to /api/* on this origin, which Next.js rewrites to the Go service. Same-origin is
 * what carries the session cookie, including on the <img> requests that fetch renditions
 * (research.md Decision 2).
 */
const API = '/api'

export interface ListParams {
  status?: CollectionStatus
  /**
   * A series to narrow to, or the empty string for the pieces recording none.
   *
   * `undefined` and `''` are different requests, and `if (params.series)` cannot tell them apart.
   * The empty string is the filter behind the statistics' unspecified group; dropping it sends that
   * group's link to the whole vault, which looks like a working page with the wrong contents
   * (008 FR-031, SC-010).
   */
  series?: string
  /** A manufacturer, with exactly the semantics of `series` above. */
  manufacturer?: string
  /** A search (feature 011). */
  q?: string
  /** The order (feature 011). A cursor is only valid with the order it came from. */
  sort?: SortOrder
  cursor?: string
  limit?: number
}

export async function listCollectibles(
  params: ListParams = {},
  init?: RequestInit,
): Promise<CollectionPage> {
  const query = new URLSearchParams()
  if (params.status) query.set('status', params.status)
  if (params.series !== undefined) query.set('series', params.series)
  if (params.manufacturer !== undefined) query.set('manufacturer', params.manufacturer)
  if (params.q) query.set('q', params.q)
  if (params.sort) query.set('sort', params.sort)
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
 * Move a collectible to the Trash, where it is kept for 30 days (feature 010).
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
