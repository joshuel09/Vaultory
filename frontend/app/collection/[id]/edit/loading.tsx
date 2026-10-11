/**
 * Shown while a collectible is being read for editing (FR-036).
 *
 * The shape of the form rather than a spinner: the collector is about to see fields, so the page
 * should not jump when they arrive.
 */
export default function EditCollectibleLoading() {
  return (
    <div className="mx-auto w-full max-w-3xl animate-pulse px-4 py-10 sm:px-6" aria-hidden="true">
      <div className="mb-8 h-4 w-40 rounded bg-surface-raised" />
      <div className="mb-10 space-y-3">
        <div className="h-8 w-2/3 rounded bg-surface-raised" />
        <div className="h-4 w-full rounded bg-surface-raised" />
      </div>
      <div className="space-y-5">
        <div className="h-10 w-full rounded bg-surface-raised" />
        <div className="h-10 w-full rounded bg-surface-raised" />
        <div className="h-28 w-[5.6rem] rounded-lg bg-surface-raised" />
        <div className="grid gap-5 sm:grid-cols-2">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="h-10 rounded bg-surface-raised" />
          ))}
        </div>
      </div>
      <span className="sr-only">Loading this collectible…</span>
    </div>
  )
}
