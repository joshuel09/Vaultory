import { cn } from '@/lib/cn'
import { STATUS_LABELS, type CollectionStatus } from '@/lib/api/types'

/**
 * Each status carries a glyph as well as a tint.
 *
 * FR-044: status must be conveyed by more than colour. A collector who cannot distinguish amber
 * from green still reads "Preordered", and the glyph gives a second cue at a glance.
 */
const GLYPHS: Record<CollectionStatus, string> = {
  owned: '◆',
  preordered: '◷',
  wishlist: '☆',
  sold: '→',
}

const TINTS: Record<CollectionStatus, string> = {
  owned: 'bg-accent/15 text-accent border-accent/30',
  preordered: 'bg-ink/10 text-ink-muted border-edge',
  wishlist: 'bg-ink/5 text-ink-faint border-edge',
  sold: 'bg-success/15 text-success border-success/30',
}

export function StatusBadge({ status, className }: { status: CollectionStatus; className?: string }) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium',
        TINTS[status],
        className,
      )}
    >
      <span aria-hidden="true">{GLYPHS[status]}</span>
      {STATUS_LABELS[status]}
    </span>
  )
}
