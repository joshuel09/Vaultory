import Link from 'next/link'
import { redirect } from 'next/navigation'
import { AuthShell } from '@/components/auth/AuthShell'
import { RegisterForm } from '@/components/auth/RegisterForm'
import { safeRedirect } from '@/lib/safe-redirect'
import { currentCollector } from '@/lib/session'

export const metadata = { title: 'Create your vault — Vaultory' }

export default async function RegisterPage({
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
      title="Create your vault"
      lead="A private place for everything you collect. No profiles, no feeds."
      footer={
        <>
          Already have a vault?{' '}
          <Link href="/sign-in" className="text-accent hover:underline">
            Sign in
          </Link>
        </>
      }
    >
      <RegisterForm next={destination} />
    </AuthShell>
  )
}
