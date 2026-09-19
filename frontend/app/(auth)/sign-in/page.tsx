import Link from 'next/link'
import { AuthShell } from '@/components/auth/AuthShell'
import { SignInForm } from '@/components/auth/SignInForm'
import { safeRedirect } from '@/lib/safe-redirect'

export const metadata = { title: 'Sign in — Vaultory' }

export default async function SignInPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>
}) {
  const params = await searchParams
  const raw = typeof params.next === 'string' ? params.next : null

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
      <SignInForm next={safeRedirect(raw)} />
    </AuthShell>
  )
}
