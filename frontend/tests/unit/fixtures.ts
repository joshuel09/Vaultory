import type { Collectible, CollectionStatus } from '@/lib/api/types'

/** A collectible with only what the spec requires, plus whatever a test needs on top. */
export function aCollectible(overrides: Partial<Collectible> = {}): Collectible {
  return {
    id: '6f9619ff-8b86-d011-b42d-00cf4fc964ff',
    name: 'Kaiju Sentinel',
    collectionStatus: 'owned' as CollectionStatus,
    character: null,
    series: null,
    manufacturer: null,
    category: null,
    scale: null,
    edition: null,
    purchasePrice: null,
    purchaseDate: null,
    releaseDate: null,
    notes: null,
    image: null,
    createdAt: '2026-09-12T10:00:00Z',
    ...overrides,
  }
}

export const anImage = {
  id: '11111111-2222-4333-8444-555555555555',
  renditionUrl: '/api/images/11111111-2222-4333-8444-555555555555/rendition',
  width: 800,
  height: 1000,
}
