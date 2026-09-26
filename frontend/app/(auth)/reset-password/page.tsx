import Link from 'next/link'
import { AuthShell } from '@/components/auth/AuthShell'
import { ResetPasswordForm } from '@/components/auth/ResetPasswordForm'

export const metadata = { title: 'Choose a new password — Vaultory' }

export default async function ResetPasswordPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>
}) {
  const params = await searchParams
  const token = typeof params.token === 'string' ? params.token : null
  const email = typeof params.email === 'string' ? params.email : null
  const failed = typeof params.error === 'string'

  if (!token || failed) {
    return (
      <AuthShell
        title="That link did not work"
        lead="It may have expired, been used already, or been replaced by a newer one."
        footer={
          <Link href="/forgot-password" className="text-accent hover:underline">
            Ask for a new link
          </Link>
        }
      >
        <p data-testid="reset-link-invalid" className="text-sm text-ink-muted">
          Reset links last one hour and work once. Asking for a new one is safe — your password has
          not changed.
        </p>
      </AuthShell>
    )
  }

  return (
    <AuthShell
      title="Choose a new password"
      lead="At least 12 characters. Every other device will be signed out."
      footer={
        <Link href="/sign-in" className="text-accent hover:underline">
          Back to sign in
        </Link>
      }
    >
      <ResetPasswordForm token={token} email={email} />
    </AuthShell>
  )
}
