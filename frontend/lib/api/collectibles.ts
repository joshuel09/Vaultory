import { toApiError } from './errors'
import type { AddCollectibleRequest, Collectible, CollectionPage, CollectionStatus } from './types'

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
