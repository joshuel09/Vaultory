import Link from 'next/link'
import { buttonClasses } from '@/components/ui/button'

/**
 * A vault with nothing in it yet (FR-041).
 *
 * Distinct from the no-results state in both words and shape: this one is an invitation, and it
 * offers the one action that resolves it.
 */
export function EmptyCollection() {
  return (
    <div
      data-testid="empty-collection"
      className="flex flex-col items-center justify-center rounded-card border border-dashed border-edge px-6 py-20 text-center"
    >
      <span aria-hidden="true" className="mb-5 text-4xl text-ink-faint/60">
        ◇
      </span>
      <h2 className="text-lg font-medium text-ink">Your vault is empty</h2>
      <p className="mt-2 max-w-sm text-sm text-ink-muted">
        Add your first collectible and it will appear here, photograph and all.
      </p>
      <Link href="/collection/new" className={buttonClasses({ className: 'mt-6' })}>
        Add a collectible
      </Link>
    </div>
  )
}
