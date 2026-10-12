'use client'

import { Button } from '@/components/ui/button'
import { STATUS_LABELS, type Collectible } from '@/lib/api/types'

interface Props {
  current: Collectible
  /** Replace what the collector typed with the values shown here. Their choice, never automatic. */
  onUseCurrent: () => void
}

/**
 * A save refused because the collectible changed first (FR-027).
 *
 * Deliberately not an error state. Nothing went wrong and nothing was lost: the collector's edit
 * is still in the form in front of them, and the collectible is intact. What they need is to see
 * what it now says and decide again.
 *
 * The current values are shown rather than merely described, because "it changed" without saying
 * how leaves the collector with no basis for deciding. Adopting them is a button they press, never
 * something that happens to them — replacing what somebody typed without being asked is the same
 * silent overwrite this whole mechanism exists to prevent.
 */
export function VersionConflictNotice({ current, onUseCurrent }: Props) {
  const rows: Array<[string, string | null]> = [
    ['Name', current.name],
    ['Status', STATUS_LABELS[current.collectionStatus]],
    ['Character', current.character ?? null],
    ['Series', current.series ?? null],
    ['Manufacturer', current.manufacturer ?? null],
    ['Purchase price', current.purchasePrice ?? null],
    ['Purchase date', current.purchaseDate ?? null],
    ['Notes', current.notes ?? null],
  ]

  return (
    <div
      data-testid="version-conflict"
      role="alert"
      className="space-y-4 rounded-lg border border-warning/40 bg-warning/10 px-4 py-4 text-sm text-ink"
    >
      <div className="space-y-1">
        <p className="font-medium">This collectible changed since you opened it.</p>
        <p className="text-ink-muted">
          Nothing has been lost — your changes are still below, and the collectible is untouched.
          Here is how it reads now.
        </p>
      </div>

      <dl className="grid gap-x-6 gap-y-1.5 sm:grid-cols-[auto_1fr]">
        {rows
          .filter(([, value]) => value !== null && value !== '')
          .map(([label, value]) => (
            <div key={label} className="contents">
              <dt className="text-xs uppercase tracking-wide text-ink-faint">{label}</dt>
              <dd className="text-ink">{value}</dd>
            </div>
          ))}
      </dl>

      <div className="flex flex-wrap items-center gap-3 pt-1">
        <Button type="button" variant="secondary" onClick={onUseCurrent}>
          Use these values
        </Button>
        <p className="text-xs text-ink-faint">
          Or keep editing below and save again — your version will be the one that stands.
        </p>
      </div>
    </div>
  )
}
