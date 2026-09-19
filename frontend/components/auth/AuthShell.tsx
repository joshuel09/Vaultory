import Link from 'next/link'

/**
 * The frame both authentication pages share.
 *
 * Built from the same tokens as every other page, so dark and light work from one definition and a
 * hard-coded colour here would be a defect (Constitution I).
 */
export function AuthShell({
  title,
  lead,
  children,
  footer,
}: {
  title: string
  lead: string
  children: React.ReactNode
  footer: React.ReactNode
}) {
  return (
    <div className="flex min-h-dvh flex-col bg-surface">
      <header className="mx-auto w-full max-w-6xl px-4 py-5 sm:px-6">
        <Link href="/" className="text-sm font-semibold tracking-tight text-ink">
          Vault<span className="text-accent">ory</span>
        </Link>
      </header>

      <main className="mx-auto flex w-full max-w-md flex-1 flex-col justify-center px-4 py-10 sm:px-6">
        <h1 className="text-2xl font-semibold tracking-tight text-ink">{title}</h1>
        <p className="mt-1.5 text-sm text-ink-muted">{lead}</p>
        <div className="mt-8">{children}</div>
        <p className="mt-6 text-sm text-ink-muted">{footer}</p>
      </main>
    </div>
  )
}
