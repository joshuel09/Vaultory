import type { GroupSummary } from '@/lib/api/types'
import { galleryHref, type GalleryFilter } from '@/lib/gallery-link'

/** The two dimensions a breakdown can group by. */
export type GroupDimension = 'series' | 'manufacturer'

/**
 * The gallery filter that reproduces one breakdown group, or null when no filter can.
 *
 * Decided from `kind`, never from the label. An unspecified group whose label read "Unspecified"
 * would otherwise be indistinguishable from a collector who typed that word as a manufacturer —
 * which is precisely why the contract carries `kind` at all.
 *
 * The remainder has no filter. It means "everything past the eighth largest group", which is not a
 * question the gallery can be asked; a link that quietly returned something else would be a worse
 * answer than no link.
 */
export function filterForGroup(
  dimension: GroupDimension,
  group: GroupSummary,
): GalleryFilter | null {
  switch (group.kind) {
    case 'named':
      return { [dimension]: group.label }
    case 'unspecified':
      return { [dimension]: '' }
    default:
      return null
  }
}

export function hrefForGroup(dimension: GroupDimension, group: GroupSummary): string | null {
  const filter = filterForGroup(dimension, group)
  return filter === null ? null : galleryHref(filter)
}
