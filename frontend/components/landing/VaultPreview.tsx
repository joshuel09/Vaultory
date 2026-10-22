import { StatusBadge } from '@/components/collection/StatusBadge'
import type { CollectionStatus } from '@/lib/api/types'

/**
 * A still life of the gallery, for people who have not signed in.
 *
 * Deliberately not real data and deliberately not a screenshot: it is the same card geometry the
 * collection view uses — a 4:5 portrait frame — so what a visitor sees here is what they get
 * (FR-005). No request is made; this is presentation only (FR-010).
 *
 * The photographs are real, which the glyph placeholders they replaced were not. FR-005 asks for
 * the gallery to be the centrepiece, and four gradients with a diamond in the middle showed the
 * geometry without showing the point of it.
 *
 * **Every one of them is deliberately unbranded**, and that constraint is load-bearing rather than
 * fussy. The Unsplash licence covers the photograph; it does not and cannot cover a character
 * someone else owns. A photograph of a Spider-Man figure carries Marvel's rights and the sculptor's
 * as well as the photographer's, and this is marketing imagery on a public repository, which is the
 * least defensible place to get that wrong. Tin robots, artist mannequins and blank articulated
 * bodies carry no such claim.
 *
 * Worth knowing if these are ever swapped: alt text is not a safe filter. Of six candidates picked
 * by description, four turned out to be Buzz Lightyear, two Stormtroopers, Woody, and an Ant-Man —
 * and that last one's description said "no identifiable franchise". They were caught by looking at
 * them. Look at any replacement.
 */
const SHOWCASE: {
  name: string
  detail: string
  status: CollectionStatus
  src: string
  alt: string
}[] = [
  {
    name: 'Kaiju Sentinel',
    detail: 'Apex Studio · 1/4 scale',
    status: 'owned',
    src: '/landing/tin-robot.jpg',
    alt: 'A blue vintage tin robot with an orange dial on its chest, lit against warm bokeh',
  },
  {
    name: 'Hollow Empress',
    detail: 'Nightfall · Deluxe',
    status: 'preordered',
    src: '/landing/wooden-mannequin.jpg',
    alt: 'A wooden artist’s mannequin posed mid-stride on a turned wooden base',
  },
  {
    name: 'Ferrous Saint',
    detail: 'Ironworks · 1/6 scale',
    status: 'wishlist',
    src: '/landing/clockwork-robot.jpg',
    alt: 'A silver clockwork tin robot with red eyes and a wind-up key',
  },
  {
    name: 'Tidebreaker',
    detail: 'Deep Current · Exclusive',
    status: 'sold',
    src: '/landing/articulated-figures.jpg',
    alt: 'Two grey articulated artist figures seated back to back',
  },
]

export function VaultPreview() {
  return (
    <ul
      aria-label="An example of a collection gallery"
      className="grid grid-cols-2 gap-3 sm:gap-4 lg:grid-cols-4"
    >
      {SHOWCASE.map((item, i) => (
        <li
          key={item.name}
          /*
           * group/rise: the card reveals in sequence, then behaves like a case you lean toward.
           * The lift and the shadow are transform and shadow only — neither reflows the grid, so
           * hovering one card cannot nudge the other three.
           */
          style={{ animationDelay: `${i * 90}ms` }}
          className="rise group overflow-hidden rounded-xl border border-edge bg-surface-raised shadow-sm transition-[transform,box-shadow,border-color] duration-500 ease-out hover:-translate-y-1.5 hover:border-accent/40 hover:shadow-lg"
        >
          {/* The same 4:5 frame every real card uses, so nothing is cropped or letterboxed. */}
          <div className="relative aspect-[4/5] overflow-hidden bg-surface">
            {/*
              A plain <img>, like the gallery's own cards: next/image is configured unoptimized for
              this application, so it would add a wrapper and no benefit. These four are the only
              images on a page that must render for someone who has never signed in, so they are
              eager rather than lazy — a landing page that pops in as it loads reads as unfinished.
            */}
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={item.src}
              alt={item.alt}
              className="absolute inset-0 h-full w-full object-cover transition-transform duration-700 ease-out group-hover:scale-[1.06]"
            />
            {/* Light crossing the glass. Decorative, and transform-only so it cannot reflow. */}
            <div aria-hidden="true" className="sheen pointer-events-none absolute inset-0" />
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
