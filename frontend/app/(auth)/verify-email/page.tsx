import Link from 'next/link'
import { AuthShell } from '@/components/auth/AuthShell'
import { buttonClasses } from '@/components/ui/button'
import { currentCollector } from '@/lib/session'

export const metadata = { title: 'Email verified — Vaultory' }

/**
 * Where a verification link lands (FR-002, FR-002a).
 *
 * Better Auth has already done the work by the time anyone arrives here — it verifies the token and
 * redirects. This page confirms it and sends the collector onward.
 *
 * It deliberately does not sign anyone in. Verification links are usually opened on a phone or in
 * a different browser, and the obvious convenience would turn a 24-hour email into a way into a
 * vault: proving you can read an inbox is not proving you know a password.
 */
export default async function VerifyEmailPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>
}) {
  const params = await searchParams
  const failed = typeof params.error === 'string'
  const collector = await currentCollector()

  if (failed) {
    return (
      <AuthShell
        title="That link did not work"
        lead="It may have expired, or it may already have been used."
        footer={
          <>
            Signed in?{' '}
            <Link href="/collection" className="text-accent hover:underline">
              Open your vault
            </Link>{' '}
            and ask for a new one.
          </>
        }
      >
        <p data-testid="verify-failed" className="text-sm text-ink-muted">
          Verification links last 24 hours. Open your vault and choose “send it again” to get a
          fresh one — nothing has been lost in the meantime.
        </p>
      </AuthShell>
    )
  }

  return (
    <AuthShell
      title="Address verified"
      lead="Thank you — we can reach you if you ever need to recover your vault."
      footer={
        collector ? (
          <>Your collection is ready.</>
        ) : (
          <>Verifying confirms your address. It does not sign you in.</>
        )
      }
    >
      <div data-testid="verify-succeeded" className="space-y-4">
        <p className="text-sm text-ink-muted">
          {collector
            ? 'You are signed in on this device.'
            : 'Sign in to open your vault on this device.'}
        </p>
        <Link
          href={collector ? '/collection' : '/sign-in'}
          className={buttonClasses({ size: 'lg' })}
        >
          {collector ? 'Open my vault' : 'Sign in'}
        </Link>
      </div>
    </AuthShell>
  )
}
