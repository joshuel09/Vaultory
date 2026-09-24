'use client'

import { useState } from 'react'
import { authClient } from '@/lib/auth-client'

/**
 * Tells a collector their address is unverified, and offers to send the message again (FR-005).
 *
 * It informs; it does not block. An unverified collector still has their vault — verification gates
 * *recovery*, not access, and holding a collection hostage to an email round trip would punish the
 * collector for a risk that is ours to manage.
 */
export function UnverifiedNotice({ email }: { email: string }) {
  const [state, setState] = useState<'idle' | 'sending' | 'sent' | 'failed'>('idle')

  return (
    <div
      data-testid="unverified-notice"
      className="border-b border-edge bg-accent/10"
      role="status"
    >
      <div className="mx-auto flex w-full max-w-6xl flex-wrap items-center gap-x-3 gap-y-1 px-4 py-2.5 sm:px-6">
        <p className="text-xs text-ink-muted">
          {state === 'sent'
            ? `Sent. Check ${email} — the link lasts 24 hours.`
            : state === 'failed'
              ? 'That did not send. Please try again shortly.'
              : `${email} is not verified yet. Verify it so you can recover your vault if you forget your password.`}
        </p>
        {state !== 'sent' && (
          <button
            type="button"
            data-testid="resend-verification"
            disabled={state === 'sending'}
            className="text-xs font-medium text-accent underline-offset-2 hover:underline disabled:opacity-50"
            onClick={async () => {
              setState('sending')
              try {
                // Resending issues a fresh link. Any previous one keeps working until it expires,
                // because verification tokens are stateless JWTs and there is nothing stored to
                // invalidate. That is decided, not overlooked — research Decision 3. Reset links
                // are different and *are* invalidated, because those grant access.
                await authClient.sendVerificationEmail({ email, callbackURL: '/verify-email' })
                setState('sent')
              } catch {
                setState('failed')
              }
            }}
          >
            {state === 'sending' ? 'Sending…' : 'Send it again'}
          </button>
        )}
      </div>
    </div>
  )
}
