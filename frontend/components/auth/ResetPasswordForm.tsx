'use client'

import { useRouter } from 'next/navigation'
import { useState } from 'react'
import { authClient } from '@/lib/auth-client'
import { Button } from '@/components/ui/button'
import { Field, Input } from '@/components/ui/field'

export function ResetPasswordForm({ token, email }: { token: string; email: string | null }) {
  const router = useRouter()
  const [password, setPassword] = useState('')
  const [fieldError, setFieldError] = useState<string | null>(null)
  const [formError, setFormError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  return (
    <form
      noValidate
      data-testid="reset-form"
      className="space-y-5"
      onSubmit={async (event) => {
        event.preventDefault()
        if (busy) return

        // Refused here, before the link is spent (FR-013). A password that is too short must not
        // cost somebody their one-use link.
        if (password.length < 12) {
          setFieldError('Passwords must be at least 12 characters.')
          return
        }
        setFieldError(null)
        setFormError(null)
        setBusy(true)

        try {
          const result = await authClient.resetPassword({ token, newPassword: password })
          if (result?.error) {
            setFormError(
              result.error.message ??
                'That link is no longer valid. Ask for a new one and try again.',
            )
            return
          }

          /*
           * Sign in with the password they have just chosen (FR-014).
           *
           * Better Auth does not do this — resetPassword returns and leaves them signed out,
           * staring at a sign-in form immediately after proving their identity. They demonstrably
           * know this password: they typed it a moment ago.
           *
           * It runs after resetPassword, which is what revokes the other sessions. Doing it the
           * other way round would create a session and then delete it.
           *
           * This is an ordinary session the Go service verifies on the next request. That matters:
           * a recovery flow must not establish access by any route the backend does not check.
           */
          if (email) {
            const signedIn = await authClient.signIn.email({ email, password })
            if (signedIn?.error) {
              /*
               * The password *was* changed — resetPassword returned without an error above, and
               * the link is spent. Only the convenience sign-in failed.
               *
               * Checking this is not defensive tidiness. The result used to be discarded, and
               * pushing to /collection without a session sends the collector through the
               * middleware to /sign-in?next=/collection, where they are shown a sign-in form and
               * told nothing at all — immediately after successfully resetting their password.
               * The likeliest cause is the sign-in rate limit, which is ten per address per
               * fifteen minutes: somebody who mistyped their old password a few times before
               * giving up and resetting it arrives here with that budget already spent. Confirmed
               * against the running stack by exhausting the limiter (issue #41).
               *
               * Reported as throttling or not, because the two need different things from the
               * collector: one is "sign in", the other is "wait, then sign in". Telling somebody
               * to sign in when the next attempt will also be refused is worse than saying
               * nothing.
               */
              const throttled = signedIn.error.status === 429
              router.push(`/sign-in?reset=${throttled ? 'throttled' : 'done'}`)
            } else {
              router.push('/collection')
            }
          } else {
            // No address to sign in as — the link was hand-assembled rather than followed from a
            // message. Say what happened rather than dropping them somewhere with no session.
            router.push('/sign-in?reset=done')
          }
          router.refresh()
        } catch {
          setFormError('Vaultory could not be reached. Please try again.')
        } finally {
          setBusy(false)
        }
      }}
    >
      {formError && (
        <div
          data-testid="form-error"
          role="alert"
          className="rounded-lg border border-danger/40 bg-danger/10 px-4 py-3 text-sm text-ink"
        >
          {formError}
        </div>
      )}

      <Field
        label="New password"
        required
        hint="At least 12 characters."
        error={fieldError ?? undefined}
      >
        {({ id, describedBy, invalid }) => (
          <Input
            id={id}
            type="password"
            autoComplete="new-password"
            autoFocus
            value={password}
            aria-invalid={invalid || undefined}
            aria-describedby={describedBy}
            onChange={(e) => setPassword(e.target.value)}
          />
        )}
      </Field>

      <Button type="submit" size="lg" className="w-full" disabled={busy}>
        {busy ? 'Saving…' : 'Set my new password'}
      </Button>
    </form>
  )
}
