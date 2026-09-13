import { StatusBadge } from './StatusBadge'
import { ImagePlaceholder } from './ImagePlaceholder'
import type { Collectible } from '@/lib/api/types'

/**
 * One collectible in the gallery.
 *
 * The image is the dominant element and fills a 4:5 frame identical for every card, matching the
 * rendition geometry exactly, so a panorama and a tall statue sit side by side without the
 * collector cropping either (FR-014, FR-030).
 */
export function CollectibleCard({ collectible }: { collectible: Collectible }) {
  const { name, collectionStatus, image, series, manufacturer } = collectible
  // Below the image, the most identifying detail a collector has recorded.
  const subtitle = series ?? manufacturer ?? null

  return (
    <article
      data-testid="collectible-card"
      className="group overflow-hidden rounded-card border border-edge bg-surface shadow-card transition-shadow hover:shadow-card-hover"
    >
      <div className="relative aspect-collectible w-full overflow-hidden bg-surface-raised">
        {image ? (
          /*
           * A plain <img>, deliberately. The rendition is already exactly 800x1000 and is served
           * from an authorized path; Next's optimizer would re-fetch it server-side without the
           * collector's session and get a 404 (research.md Decision 2, FR-015).
           *
           * object-cover fills the frame; since the rendition already matches the frame's ratio,
           * nothing is actually trimmed here.
           */
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={image.renditionUrl}
            alt={`Photograph of ${name}`}
            width={image.width}
            height={image.height}
            loading="lazy"
            decoding="async"
            className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-[1.03]"
          />
        ) : (
          <ImagePlaceholder name={name} />
        )}
      </div>

      <div className="space-y-2 p-3.5">
        <h3 className="line-clamp-2 text-sm font-medium leading-snug text-ink" title={name}>
          {name}
        </h3>
        {subtitle && <p className="line-clamp-1 text-xs text-ink-muted">{subtitle}</p>}
        <StatusBadge status={collectionStatus} />
      </div>
    </article>
  )
}
