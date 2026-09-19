'use client'

import { signUp } from '@/lib/auth-client'
import { AuthForm } from './AuthForm'

export function RegisterForm({ next }: { next: string }) {
  return (
    <AuthForm
      mode="register"
      submitLabel="Create my vault"
      next={next}
      onSubmit={({ email, password }) =>
        // Better Auth requires a name. Vaultory does not ask for one — a collection is private, so
        // there is nobody to display it to — so the email's local part stands in, matching what the
        // database trigger uses for display_name.
        signUp.email({ email, password, name: email.split('@')[0] ?? email })
      }
    />
  )
}
