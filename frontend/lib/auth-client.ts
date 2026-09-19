'use client'

import { createAuthClient } from 'better-auth/react'

/**
 * Browser-side helpers. These talk to /api/auth/* on this origin, which is same-origin and so
 * carries the session cookie automatically.
 */
export const authClient = createAuthClient()

export const { signIn, signUp, signOut, useSession } = authClient
