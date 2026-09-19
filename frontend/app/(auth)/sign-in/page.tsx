import Link from 'next/link'
import { redirect } from 'next/navigation'
import { AuthShell } from '@/components/auth/AuthShell'
import { SignInForm } from '@/components/auth/SignInForm'
import { safeRedirect } from '@/lib/safe-redirect'
import { currentCollector } from '@/lib/session'

export const metadata = { title: 'Sign in — Vaultory' }

export default async function SignInPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>
}) {
  const params = await searchParams
  const raw = typeof params.next === 'string' ? params.next : null
  const destination = safeRedirect(raw)

  // Already signed in? There is nothing here for you. Offering a sign-in form to someone who is
  // signed in invites them to authenticate twice and leaves them wondering which account they are
  // now using.
  const collector = await currentCollector()
  if (collector) redirect(destination)


  return (
    <AuthShell
      title="Sign in"
      lead="Your collection is where you left it."
      footer={
        <>
          No vault yet?{' '}
          <Link href="/register" className="text-accent hover:underline">
            Create one
          </Link>
        </>
      }
    >
      <SignInForm next={destination} />
    </AuthShell>
  )
}
