'use client'

import { useState } from 'react'
import { authClient } from '@/lib/auth-client'
import { Button } from '@/components/ui/button'
import { Field, Input } from '@/components/ui/field'

/**
 * Asking for a reset link (FR-007, FR-008).
 *
 * The answer is the same whether or not the address has an account. That is not vagueness for its
 * own sake: a form that says "no such account" is a form that tells anyone which addresses have a
 * vault here, and a collection is private.
 */
export function ForgotPasswordForm() {
  const [email, setEmail] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [sent, setSent] = useState(false)

  if (sent) {
    return (
      <div data-testid="reset-requested" className="space-y-3" role="status">
        <p className="text-sm text-ink">
          If that address has a Vaultory account, a reset link is on its way to it.
        </p>
        <p className="text-sm text-ink-muted">
          The link works for one hour and can be used once. Nothing has changed about your account
          yet.
        </p>
      </div>
    )
  }

  return (
    <form
      noValidate
      data-testid="forgot-form"
      className="space-y-5"
      onSubmit={async (event) => {
        event.preventDefault()
        if (busy) return

        const address = email.trim().toLowerCase()
        if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(address)) {
          setError('That does not look like an email address.')
          return
        }

        setError(null)
        setBusy(true)
        try {
          await authClient.requestPasswordReset({ email: address, redirectTo: '/reset-password' })
          // Shown whatever happened. The response from the server is deliberately the same for an
          // address with an account and one without, and this must not undo that.
          setSent(true)
        } catch {
          setSent(true)
        } finally {
          setBusy(false)
        }
      }}
    >
      <Field label="Email" required error={error ?? undefined}>
        {({ id, describedBy, invalid }) => (
          <Input
            id={id}
            type="email"
            autoComplete="email"
            autoFocus
            value={email}
            placeholder="you@example.com"
            aria-invalid={invalid || undefined}
            aria-describedby={describedBy}
            onChange={(e) => setEmail(e.target.value)}
          />
        )}
      </Field>

      <Button type="submit" size="lg" className="w-full" disabled={busy}>
        {busy ? 'Sending…' : 'Send me a link'}
      </Button>
    </form>
  )
}
