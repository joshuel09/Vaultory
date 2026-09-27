import Link from 'next/link'
import { AuthShell } from '@/components/auth/AuthShell'
import { ForgotPasswordForm } from '@/components/auth/ForgotPasswordForm'

export const metadata = { title: 'Reset your password — Vaultory' }

export default function ForgotPasswordPage() {
  return (
    <AuthShell
      title="Reset your password"
      lead="We will send a link to the address on your account."
      footer={
        <>
          Remembered it?{' '}
          <Link href="/sign-in" className="text-accent hover:underline">
            Sign in
          </Link>
        </>
      }
    >
      <ForgotPasswordForm />
    </AuthShell>
  )
}
