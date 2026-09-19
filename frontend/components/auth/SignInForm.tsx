'use client'

import { signIn } from '@/lib/auth-client'
import { AuthForm } from './AuthForm'

export function SignInForm({ next }: { next: string }) {
  return (
    <AuthForm
      mode="sign-in"
      submitLabel="Sign in"
      next={next}
      onSubmit={({ email, password }) => signIn.email({ email, password })}
    />
  )
}
