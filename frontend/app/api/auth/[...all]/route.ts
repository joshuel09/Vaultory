import { toNextJsHandler } from 'better-auth/next-js'
import { auth } from '@/lib/auth'

/**
 * Better Auth's own routes: sign-up, sign-in, sign-out, session.
 *
 * Deliberately absent from the OpenAPI contract. That contract is the source of truth for the
 * frontend/backend seam, and these do not cross it — they are the frontend talking to itself.
 * Writing them into it would assert that the Go service serves them, which is false.
 */
export const { GET, POST } = toNextJsHandler(auth)
