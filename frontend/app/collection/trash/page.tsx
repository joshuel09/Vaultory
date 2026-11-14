import Link from 'next/link'
import { cookies } from 'next/headers'
import { redirect } from 'next/navigation'
import { TrashList } from '@/components/collection/TrashList'
import type { TrashPage } from '@/lib/api/types'

/**
 * The Trash (feature 010). Rendered on the server like the gallery; only restoring, deleting and
 * loading more are interactive.
 */

// Per collector, and changes with every deletion and restore.
export const dynamic = 'force-dynamic'

const BACKEND = process.env.VAULTORY_BACKEND_ORIGIN ?? 'http://127.0.0.1:8080'

async function fetchFirstPage(): Promise<TrashPage> {
  // Server-side the rewrite is not in play, so this reaches Go directly and forwards the cookie.
  const cookieHeader = (await cookies()).toString()
  const response = await fetch(new URL('/api/trash', BACKEND), {
    headers: cookieHeader ? { cookie: cookieHeader } : {},
    cache: 'no-store',
  })
  if (response.status === 401) {
    redirect('/sign-in?next=/collection/trash')
  }
  if (!response.ok) {
    throw new Error(`trash request failed with ${response.status}`)
  }
  return (await response.json()) as TrashPage
}

export default async function TrashPageRoute() {
  const page = await fetchFirstPage()
  return (
    <main className="mx-auto w-full max-w-3xl px-4 py-8 sm:px-6">
      <header className="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-ink">Trash</h1>
          <p className="mt-1 text-sm text-ink-muted">
            Deleted collectibles are kept here for 30 days, then removed for good.
          </p>
        </div>
        <Link
          href="/collection"
          className="text-sm font-medium text-ink-muted underline-offset-4 hover:text-ink hover:underline"
        >
          Back to your vault
        </Link>
      </header>
      <TrashList initialItems={page.items} initialCursor={page.nextCursor ?? null} />
    </main>
  )
}
