import Link from 'next/link'
import { buttonClasses } from '@/components/ui/button'
import { currentCollector } from '@/lib/session'
import { VaultPreview } from '@/components/landing/VaultPreview'
import { STATUS_LABELS, COLLECTION_STATUSES } from '@/lib/api/types'

/**
 * Vaultory's landing page (feature 003).
 *
 * Public, sessionless, and static: it renders and decides nothing, and makes no request to /api/*
 * (FR-010). The call to action leads to registration; a returning collector signs in from the
 * header.
 */

export const metadata = {
  title: 'Vaultory — your collection, kept properly',
  description:
    'A private digital vault for collectors of figures, statues, comics and trading collectibles. ' +
    'Track what you own, what is on preorder, what you are hunting, and what you have sold.',
}

/** What a vault records, in the order a collector tends to know it. */
const RECORDED = [
  ['Character', 'Sentinel Prime'],
  ['Series', 'Kaiju Wars'],
  ['Manufacturer', 'Apex Studio'],
  ['Category', 'Statue'],
  ['Scale', '1/4'],
  ['Edition', 'Deluxe Exclusive'],
  ['Purchase price', '249.99'],
  ['Purchase date', 'when it arrived'],
  ['Release date', 'when it ships'],
  ['Notes', 'box condition, provenance'],
] as const

/**
 * The words that drift past under the hero.
 *
 * Invented series and makers, matching the showcase cards, rather than real brands — the same
 * reason the preview photographs are unbranded. It is texture that says "this is for figure
 * collectors" without borrowing anybody's trademark to say it.
 */
const MARQUEE = [
  'Kaiju Wars',
  'Apex Studio',
  '1/4 scale',
  'Nightfall',
  'Deluxe Exclusive',
  'Ironworks',
  'Resin statue',
  'Deep Current',
  'Preorder',
  'Vinyl',
  'Chase variant',
  'Grail',
] as const

// FR-024. Feature 004 replaced the temporary development sign-in this used to point at.
const ENTER_VAULT = '/register'

export default async function Home() {
  const collector = await currentCollector()

  return (
    <div className="min-h-dvh bg-surface">
      <header className="mx-auto flex w-full max-w-6xl items-center justify-between px-4 py-5 sm:px-6">
        <span className="text-sm font-semibold tracking-tight text-ink">
          Vault<span className="text-accent">ory</span>
        </span>
        {collector ? (
          <Link href="/collection" className={buttonClasses({ variant: 'ghost', size: 'sm' })}>
            My vault
          </Link>
        ) : (
          <Link href="/sign-in" className={buttonClasses({ variant: 'ghost', size: 'sm' })}>
            Sign in
          </Link>
        )}
      </header>

      <main>
        {/* Hero ---------------------------------------------------------------------------- */}
        {/* relative + overflow-hidden so the drifting glow cannot widen the page (SC-010). */}
        <section className="relative overflow-hidden">
          {/* Decorative only: a slow wash of the accent, as light changes over a cabinet. */}
          <div aria-hidden="true" className="vault-glow pointer-events-none absolute inset-0 -z-10" />

          <div className="mx-auto w-full max-w-6xl px-4 pb-14 pt-10 sm:px-6 sm:pb-20 sm:pt-16">
            <p className="rise mb-4 text-xs font-medium uppercase tracking-[0.2em] text-accent">
              For collectors, not inventories
            </p>
            <h1
              className="rise max-w-3xl text-4xl font-semibold leading-[1.1] tracking-tight text-ink sm:text-5xl lg:text-6xl"
              style={{ animationDelay: '80ms' }}
            >
              Your collection, kept properly.
            </h1>
            <p
              className="rise mt-5 max-w-2xl text-base leading-relaxed text-ink-muted sm:text-lg"
              style={{ animationDelay: '160ms' }}
            >
              Vaultory is a private vault for figures, statues, comics and trading collectibles.
              Every piece you own, every preorder you are waiting on, everything still on the hunt —
              held in one place and shown the way you would actually display it.
            </p>

            <div
              className="rise mt-8 flex flex-wrap items-center gap-3"
              style={{ animationDelay: '240ms' }}
            >
              <Link
                href={collector ? '/collection' : ENTER_VAULT}
                className={buttonClasses({ size: 'lg' })}
              >
                {collector ? 'Open my vault' : 'Create my vault'}
              </Link>
              {!collector && (
                <span className="text-xs text-ink-faint">
                  Free, private, and yours. No profiles, no feeds.
                </span>
              )}
            </div>
          </div>

          {/*
            The marquee: the vocabulary of the hobby, drifting past.
            
            Decorative and aria-hidden — it is texture, not information, and a screen reader
            reading the same twelve words twice would be worse than silence. The list is duplicated
            because the loop translates by exactly -50%, which is seamless only if the second half
            repeats the first.
          */}
          <div
            aria-hidden="true"
            className="relative flex overflow-hidden border-y border-edge bg-surface-raised/30 py-3"
          >
            <div className="marquee-track flex shrink-0 items-center gap-8 pr-8 sm:gap-12 sm:pr-12">
              {[...MARQUEE, ...MARQUEE].map((word, i) => (
                <span
                  key={`${word}-${i}`}
                  className="whitespace-nowrap text-xs uppercase tracking-[0.18em] text-ink-faint"
                >
                  {word}
                </span>
              ))}
            </div>
          </div>
        </section>

        {/* The gallery, which is the whole premise ----------------------------------------- */}
        {/* pt here, not only pb: the hero's bottom padding now ends above the marquee, so without
            this the strip's border sits flush against the heading. */}
        <section className="mx-auto w-full max-w-6xl px-4 pb-16 pt-14 sm:px-6 sm:pb-24 sm:pt-20">
          <div className="mb-6 flex flex-wrap items-baseline justify-between gap-2">
            <h2 className="text-xl font-semibold tracking-tight text-ink sm:text-2xl">
              A gallery, not a spreadsheet
            </h2>
            <p className="text-sm text-ink-faint">Every piece framed identically, so the shelf reads at a glance.</p>
          </div>
          <VaultPreview />
        </section>

        {/* Statuses ------------------------------------------------------------------------ */}
        <section className="border-y border-edge bg-surface-raised/40">
          <div className="mx-auto w-full max-w-6xl px-4 py-14 sm:px-6 sm:py-20">
            <h2 className="text-xl font-semibold tracking-tight text-ink sm:text-2xl">
              Four states, because collecting has four states
            </h2>
            <p className="mt-2 max-w-2xl text-sm leading-relaxed text-ink-muted">
              A collection is not only what is on the shelf. What you have ordered, what you are
              still chasing, and what you have let go are all part of the record.
            </p>
            <dl className="mt-8 grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
              {COLLECTION_STATUSES.map((status, i) => (
                <div
                  key={status}
                  className="rise rounded-xl border border-edge bg-surface p-5 transition-colors duration-300 hover:border-accent/50"
                  style={{ animationDelay: `${i * 70}ms` }}
                >
                  <dt className="text-sm font-medium text-ink">{STATUS_LABELS[status]}</dt>
                  <dd className="mt-1.5 text-sm leading-relaxed text-ink-muted">
                    {status === 'owned' && 'On your shelf, in your hands, yours.'}
                    {status === 'preordered' && 'Paid for and waiting on a release date.'}
                    {status === 'wishlist' && 'Still hunting. Kept so you do not buy it twice.'}
                    {status === 'sold' && 'Gone, but part of your history all the same.'}
                  </dd>
                </div>
              ))}
            </dl>
          </div>
        </section>

        {/* What it records ----------------------------------------------------------------- */}
        <section className="mx-auto w-full max-w-6xl px-4 py-14 sm:px-6 sm:py-20">
          <div className="grid gap-10 lg:grid-cols-2 lg:gap-16">
            <div>
              <h2 className="text-xl font-semibold tracking-tight text-ink sm:text-2xl">
                As much detail as you care to keep
              </h2>
              <p className="mt-2 max-w-xl text-sm leading-relaxed text-ink-muted">
                A name and a status are all that is ever required. Everything else is there for the
                day you want it — what you paid, when it shipped, which variant it is, and the note
                you will be glad you wrote.
              </p>
              <p className="mt-4 max-w-xl text-sm leading-relaxed text-ink-muted">
                Prices are kept as exact decimals, never as floating point, because a collection's
                value should not drift by a cent.
              </p>
            </div>
            <dl className="grid grid-cols-2 gap-x-6 gap-y-3 self-start">
              {RECORDED.map(([label, example]) => (
                <div key={label} className="border-b border-edge pb-2.5">
                  <dt className="text-sm text-ink">{label}</dt>
                  <dd className="truncate text-xs text-ink-faint">{example}</dd>
                </div>
              ))}
            </dl>
          </div>
        </section>

        {/* Privacy ------------------------------------------------------------------------- */}
        <section className="border-t border-edge">
          <div className="mx-auto w-full max-w-6xl px-4 py-14 sm:px-6 sm:py-20">
            <h2 className="text-xl font-semibold tracking-tight text-ink sm:text-2xl">Yours alone</h2>
            <p className="mt-2 max-w-2xl text-sm leading-relaxed text-ink-muted">
              A vault is private. No profiles, no feeds, no browsing other people's shelves. Every
              request is scoped to the collector who made it, and that is enforced by the server and
              by the database itself — not by hiding a button.
            </p>
            <div className="mt-8">
              <Link
                href={collector ? '/collection' : ENTER_VAULT}
                className={buttonClasses({ size: 'lg' })}
              >
                {collector ? 'Open my vault' : 'Create my vault'}
              </Link>
            </div>
          </div>
        </section>
      </main>

      <footer className="border-t border-edge">
        <div className="mx-auto w-full max-w-6xl px-4 py-8 sm:px-6">
          <p className="text-xs text-ink-faint">
            Vaultory — a private vault for collectors.
          </p>
        </div>
      </footer>
    </div>
  )
}
