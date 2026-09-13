/**
 * Shown while the collection is being fetched (FR-042).
 *
 * Skeleton cards on the gallery's own grid, not a spinner and not a blank screen: the collector
 * sees the shape of what is arriving. Critically, this is never the empty state — flashing "your
 * vault is empty" at someone whose vault is full is the failure this exists to prevent.
 */
export default function Loading() {
  return (
    <main className="mx-auto w-full max-w-[110rem] px-4 py-8 sm:px-6 lg:px-10">
      <div className="mb-8 space-y-2">
        <div className="h-8 w-40 rounded-lg shimmer" />
        <div className="h-4 w-24 rounded shimmer" />
      </div>
      <div
        role="status"
        aria-live="polite"
        className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6"
      >
        <span className="sr-only">Loading your collection</span>
        {Array.from({ length: 12 }).map((_, i) => (
          <div key={i} className="overflow-hidden rounded-card border border-edge">
            <div className="aspect-collectible w-full shimmer" />
            <div className="space-y-2 p-3.5">
              <div className="h-4 w-3/4 rounded shimmer" />
              <div className="h-5 w-20 rounded-full shimmer" />
            </div>
          </div>
        ))}
      </div>
    </main>
  )
}
