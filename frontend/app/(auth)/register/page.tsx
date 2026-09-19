import Link from 'next/link'
import { AuthShell } from '@/components/auth/AuthShell'
import { RegisterForm } from '@/components/auth/RegisterForm'
import { safeRedirect } from '@/lib/safe-redirect'

export const metadata = { title: 'Create your vault — Vaultory' }

export default async function RegisterPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>
}) {
  const params = await searchParams
  const raw = typeof params.next === 'string' ? params.next : null

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
      <RegisterForm next={safeRedirect(raw)} />
    </AuthShell>
  )
}
