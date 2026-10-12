import Link from 'next/link'
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
export function CollectibleCard({
  collectible,
  returnTo = '/collection',
}: {
  collectible: Collectible
  /**
   * Where editing should send the collector back to — the gallery as they are currently looking at
   * it, status filter and all, so they are not dropped on an unfiltered one.
   */
  returnTo?: string
}) {
  const { id, name, collectionStatus, image, series, manufacturer } = collectible
  // Below the image, the most identifying detail a collector has recorded.
  const subtitle = series ?? manufacturer ?? null
  const editHref = `/collection/${id}/edit?next=${encodeURIComponent(returnTo)}`

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
        <div className="flex items-center justify-between gap-2">
          <StatusBadge status={collectionStatus} />
          {/*
            * Editing is reached from the collectible's own entry (FR-001). Deleting is not offered
            * here and must not be (FR-022a): a destructive action one stray click away in a
            * browsing context is how a collection gets damaged by accident.
            */}
          <Link
            href={editHref}
            data-testid="edit-collectible"
            // Named for a screen reader rather than with hidden text: a gallery of fifty cards
            // otherwise offers fifty links all called "Edit". aria-label keeps that distinction out
            // of the DOM's text, where it would duplicate the heading.
            aria-label={`Edit ${name}`}
            className="rounded px-1.5 py-0.5 text-xs text-ink-muted underline-offset-4 transition-colors hover:text-ink hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
          >
            Edit
          </Link>
        </div>
      </div>
    </article>
  )
}
