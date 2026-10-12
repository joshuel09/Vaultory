import Link from 'next/link'
import { buttonClasses } from '@/components/ui/button'

/**
 * The collectible is not in this collector's vault.
 *
 * Deliberately says nothing about why. It may never have existed, it may have been deleted from
 * another tab, or it may belong to somebody else — and the three are indistinguishable here on
 * purpose, because telling them apart would disclose that a stranger's collectible exists (FR-031).
 *
 * There is no Try again, because trying again cannot work.
 */
export default function CollectibleNotFound() {
  return (
    <div className="mx-auto flex w-full max-w-lg flex-col items-center gap-5 px-4 py-24 text-center">
      <div
        aria-hidden="true"
        className="flex h-16 w-16 items-center justify-center rounded-full border border-edge bg-surface-raised text-2xl"
      >
        ∅
      </div>
      <h1 className="text-xl font-semibold tracking-tight text-ink">
        That collectible is not in your vault
      </h1>
      <p className="text-sm text-ink-muted">
        It may have been deleted. Your collection is where everything you still have lives.
      </p>
      <Link href="/collection" className={buttonClasses({ size: 'lg' })}>
        Back to my collection
      </Link>
    </div>
  )
}
