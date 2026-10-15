/**
 * Shown while the figures are being fetched.
 *
 * Deliberately not a grid of zeros. A zero that will shortly become 1,250.00 is a wrong number
 * shown confidently, and a collector reading it has no way to know it is provisional (FR-038).
 */
export default function LoadingStatistics() {
  return (
    <div className="mx-auto w-full max-w-4xl px-4 py-10 sm:px-6">
      <div className="mb-10 h-9 w-64 animate-pulse rounded-lg bg-surface-raised" />
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="h-32 animate-pulse rounded-xl bg-surface-raised" />
        <div className="h-32 animate-pulse rounded-xl bg-surface-raised" />
      </div>
      <span className="sr-only" role="status">
        Working out what your collection holds…
      </span>
    </div>
  )
}
