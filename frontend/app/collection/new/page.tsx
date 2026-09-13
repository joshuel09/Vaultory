import Link from 'next/link'
import { AddCollectibleForm } from '@/components/collection/AddCollectibleForm'

export default function NewCollectiblePage() {
  return (
    <main className="mx-auto w-full max-w-3xl px-4 py-8 sm:px-6 lg:py-12">
      <nav className="mb-8">
        <Link
          href="/collection"
          className="text-sm text-ink-muted transition-colors hover:text-ink"
        >
          <span aria-hidden="true">←</span> Back to your vault
        </Link>
      </nav>

      <header className="mb-9">
        <h1 className="text-2xl font-semibold tracking-tight text-ink">Add a collectible</h1>
        <p className="mt-1.5 text-sm text-ink-muted">
          Record as much or as little as you like — you can start with just a name.
        </p>
      </header>

      <AddCollectibleForm />
    </main>
  )
}
