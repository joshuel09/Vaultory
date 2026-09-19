import Link from 'next/link'
import { SignOutButton } from '@/components/auth/SignOutButton'
import { currentCollector } from '@/lib/session'

/**
 * The bar across every vault page (FR-025).
 *
 * It exists because there was no way back out of a vault: a collector who registered landed in
 * their collection with no route home and no way to sign out. It shows which account is signed in,
 * so someone sharing a machine can tell at a glance.
 */
export async function VaultHeader() {
  const collector = await currentCollector()

  return (
    <header className="border-b border-edge">
      <div className="mx-auto flex w-full max-w-6xl items-center justify-between gap-4 px-4 py-4 sm:px-6">
        <Link href="/" className="text-sm font-semibold tracking-tight text-ink">
          Vault<span className="text-accent">ory</span>
        </Link>

        {collector ? (
          <div className="flex items-center gap-3">
            <Link
              href="/collection"
              className="text-sm text-ink-muted transition-colors hover:text-ink"
            >
              My vault
            </Link>
            <span
              data-testid="signed-in-as"
              className="hidden max-w-[16rem] truncate text-xs text-ink-faint sm:inline"
              title={collector.email}
            >
              {collector.email}
            </span>
            <SignOutButton />
          </div>
        ) : (
          <Link href="/sign-in" className="text-sm text-ink-muted transition-colors hover:text-ink">
            Sign in
          </Link>
        )}
      </div>
    </header>
  )
}
