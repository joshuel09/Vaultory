'use client'

import { useRouter } from 'next/navigation'
import { useState } from 'react'
import { signOut } from '@/lib/auth-client'
import { buttonClasses } from '@/components/ui/button'

/**
 * Signing out deletes the session row, so the next request finds nothing and is refused
 * immediately (FR-011). router.refresh() then discards the cached server render, which is what
 * stops a vault page being served from the back-forward cache afterwards.
 */
export function SignOutButton() {
  const router = useRouter()
  const [busy, setBusy] = useState(false)

  return (
    <button
      type="button"
      disabled={busy}
      data-testid="sign-out"
      className={buttonClasses({ variant: 'ghost', size: 'sm' })}
      onClick={async () => {
        setBusy(true)
        try {
          await signOut()
          router.replace('/')
          router.refresh()
        } finally {
          setBusy(false)
        }
      }}
    >
      {busy ? 'Signing out…' : 'Sign out'}
    </button>
  )
}
