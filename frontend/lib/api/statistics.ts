import { toApiError } from './errors'
import type { CollectionStatistics } from './types'

const API = '/api'

/**
 * Fetch the figures for the acting collector's vault.
 *
 * There are no parameters — not here and not in the contract. A collector identifier cannot be
 * passed to this operation, so there is no request for someone else's figures to refuse (FR-003).
 *
 * Everything that comes back is finished. Nothing in this application sums, counts, or ranks any of
 * it; that is Go's job, and a total computed here would be a business rule living in the
 * presentation layer (FR-001).
 */
export async function getCollectionStatistics(init?: RequestInit): Promise<CollectionStatistics> {
  const response = await fetch(`${API}/collectibles/statistics`, {
    ...init,
    // The figures describe a vault that changes as the collector edits it; never a cached copy
    // (FR-039).
    cache: 'no-store',
    credentials: 'same-origin',
  })
  if (!response.ok) throw await toApiError(response)
  return (await response.json()) as CollectionStatistics
}
