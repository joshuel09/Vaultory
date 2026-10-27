import { DEFAULT_SORT, type CollectionStatus, type SortOrder } from '@/lib/api/types'

/**
 * How the gallery is narrowed, and the one place a link to it is built.
 *
 * Every figure in the statistics links here, and a figure whose link disagrees with it is worse
 * than no link at all: the collector counts the pieces and finds the number was wrong. Building
 * these links in one place is what keeps the two in step (FR-030, FR-031, SC-010).
 *
 * `null` and `''` are different filters. `null` is "every series"; `''` is "the pieces recording no
 * series", which is the group the statistics label as unspecified. Writing `if (series)` collapses
 * them and sends that group's link to the whole vault — the single easiest mistake to make here,
 * and an invisible one, because the page still renders.
 */
export interface GalleryFilter {
  status?: CollectionStatus | null
  series?: string | null
  manufacturer?: string | null
  /** A search (feature 011). Unlike series, an empty search means no search, so it is dropped. */
  q?: string | null
  /** The order (feature 011). The default is dropped, so a plain gallery address stays plain. */
  sort?: SortOrder | null
}

export function galleryHref(filter: GalleryFilter = {}): string {
  const query = new URLSearchParams()
  // Explicitly against null and undefined, never truthiness: an empty value is a real filter.
  if (filter.status != null) query.set('status', filter.status)
  if (filter.series != null) query.set('series', filter.series)
  if (filter.manufacturer != null) query.set('manufacturer', filter.manufacturer)
  // Here truthiness is right: a blank search is no search, which the server agrees with.
  const q = filter.q?.trim()
  if (q) query.set('q', q)
  if (filter.sort && filter.sort !== DEFAULT_SORT) query.set('sort', filter.sort)

  const suffix = query.toString()
  return suffix ? `/collection?${suffix}` : '/collection'
}

/**
 * Whether the gallery is narrowed — and so whether nothing showing means "nothing matches" rather
 * than "nothing here". A search narrows; an order does not.
 */
export function isFiltered(filter: GalleryFilter): boolean {
  return (
    filter.status != null ||
    filter.series != null ||
    filter.manufacturer != null ||
    Boolean(filter.q?.trim())
  )
}
