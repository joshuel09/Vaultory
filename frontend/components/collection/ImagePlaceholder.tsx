import { cn } from '@/lib/cn'

/**
 * What a collectible without a photograph shows (FR-033).
 *
 * Deliberately designed rather than a broken-image glyph or an empty box: a vault with a few
 * unphotographed pieces should still look like a vault. It occupies exactly the same frame as a
 * rendition, so a mixed gallery stays on its grid.
 */
export function ImagePlaceholder({ name, className }: { name: string; className?: string }) {
  // The first letter gives each placeholder its own character, so a column of them does not read
  // as a column of identical failures.
  const initial = Array.from(name.trim())[0]?.toUpperCase() ?? '?'

  return (
    <div
      data-testid="image-placeholder"
      className={cn(
        'flex h-full w-full flex-col items-center justify-center gap-3',
        'bg-gradient-to-br from-surface-raised to-surface',
        className,
      )}
    >
      <span
        aria-hidden="true"
        className="font-sans text-5xl font-light tracking-tight text-ink-faint/60"
      >
        {initial}
      </span>
      <span className="text-[0.7rem] uppercase tracking-widest text-ink-faint/70">No photo yet</span>
    </div>
  )
}
