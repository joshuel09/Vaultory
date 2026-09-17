import { StatusBadge } from '@/components/collection/StatusBadge'
import type { CollectionStatus } from '@/lib/api/types'

/**
 * A still life of the gallery, for people who have not signed in.
 *
 * Deliberately not real data and deliberately not a screenshot: it is the same card geometry the
 * collection view uses — a 4:5 portrait frame — so what a visitor sees here is what they get
 * (FR-005). No request is made; this is presentation only (FR-010).
 */
const SHOWCASE: { name: string; detail: string; status: CollectionStatus; tone: string }[] = [
  { name: 'Kaiju Sentinel', detail: 'Apex Studio · 1/4 scale', status: 'owned', tone: 'from-accent/25 to-surface' },
  { name: 'Hollow Empress', detail: 'Nightfall · Deluxe', status: 'preordered', tone: 'from-ink/15 to-surface' },
  { name: 'Ferrous Saint', detail: 'Ironworks · 1/6 scale', status: 'wishlist', tone: 'from-accent/15 to-surface' },
  { name: 'Tidebreaker', detail: 'Deep Current · Exclusive', status: 'sold', tone: 'from-success/20 to-surface' },
]

export function VaultPreview() {
  return (
    <ul
      aria-label="An example of a collection gallery"
      className="grid grid-cols-2 gap-3 sm:gap-4 lg:grid-cols-4"
    >
      {SHOWCASE.map((item) => (
        <li
          key={item.name}
          className="overflow-hidden rounded-xl border border-edge bg-surface-raised shadow-sm"
        >
          {/* The same 4:5 frame every real card uses, so nothing is cropped or letterboxed. */}
          <div className={`relative aspect-[4/5] bg-gradient-to-br ${item.tone}`}>
            <div className="absolute inset-0 flex items-center justify-center">
              <span aria-hidden="true" className="text-4xl text-ink/25 sm:text-5xl">
                ◆
              </span>
            </div>
            <div className="absolute left-2 top-2">
              <StatusBadge status={item.status} />
            </div>
          </div>
          <div className="space-y-0.5 p-3">
            <p className="truncate text-sm font-medium text-ink">{item.name}</p>
            <p className="truncate text-xs text-ink-faint">{item.detail}</p>
          </div>
        </li>
      ))}
    </ul>
  )
}
