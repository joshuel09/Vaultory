'use client'

import { useRouter } from 'next/navigation'
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Field, Input } from '@/components/ui/field'

export interface AuthFormProps {
  mode: 'register' | 'sign-in'
  submitLabel: string
  /** Where to land afterwards. FR-022/FR-023: a remembered destination wins, the collection is the fallback. */
  next: string
  onSubmit: (values: { email: string; password: string }) => Promise<{ error?: { message?: string } | null }>
}

/**
 * The shared credential form.
 *
 * Failure reporting follows the add-collectible form (FR-026): every problem shown at once against
 * the field responsible, announced to assistive technology, and never discarding what was typed.
 * Losing a filled-in form to a failed submit is the thing this must not do.
 */
export function AuthForm({ mode, submitLabel, next, onSubmit }: AuthFormProps) {
  const router = useRouter()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  function validate() {
    const problems: Record<string, string> = {}
    if (email.trim() === '') problems.email = 'An email address is required.'
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim()))
      problems.email = 'That does not look like an email address.'
    if (password === '') problems.password = 'A password is required.'
    else if (mode === 'register' && password.length < 12)
      problems.password = 'Passwords must be at least 12 characters.'
    return problems
  }

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault()
    if (busy) return

    // Every problem at once, rather than one per attempt.
    const problems = validate()
    setFieldErrors(problems)
    setFormError(null)
    if (Object.keys(problems).length > 0) return

    setBusy(true)
    try {
      // Lower-cased so that uniqueness and sign-in are case-insensitive (FR-003).
      const result = await onSubmit({ email: email.trim().toLowerCase(), password })
      if (result?.error) {
        setFormError(result.error.message ?? 'That did not work. Please try again.')
        return
      }
      router.push(next)
      router.refresh()
    } catch {
      setFormError('Vaultory could not be reached. Please try again.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} noValidate className="space-y-5" data-testid={`${mode}-form`}>
      {formError && (
        <div
          data-testid="form-error"
          role="alert"
          className="rounded-lg border border-danger/40 bg-danger/10 px-4 py-3 text-sm text-ink"
        >
          {formError}
        </div>
      )}

      <Field label="Email" required error={fieldErrors.email}>
        {({ id, describedBy, invalid }) => (
          <Input
            id={id}
            type="email"
            autoComplete="email"
            value={email}
            placeholder="you@example.com"
            aria-invalid={invalid || undefined}
            aria-describedby={describedBy}
            onChange={(e) => setEmail(e.target.value)}
          />
        )}
      </Field>

      <Field
        label="Password"
        required
        hint={mode === 'register' ? 'At least 12 characters.' : undefined}
        error={fieldErrors.password}
      >
        {({ id, describedBy, invalid }) => (
          <Input
            id={id}
            type="password"
            autoComplete={mode === 'register' ? 'new-password' : 'current-password'}
            value={password}
            aria-invalid={invalid || undefined}
            aria-describedby={describedBy}
            onChange={(e) => setPassword(e.target.value)}
          />
        )}
      </Field>

      <Button type="submit" size="lg" className="w-full" disabled={busy}>
        {busy ? 'One moment…' : submitLabel}
      </Button>
    </form>
  )
}
