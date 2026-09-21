import { toNextJsHandler } from 'better-auth/next-js'
import { auth } from '@/lib/auth'

/**
 * Better Auth's own routes: sign-up, sign-in, sign-out, session.
 *
 * Deliberately absent from the OpenAPI contract. That contract is the source of truth for the
 * frontend/backend seam, and these do not cross it — they are the frontend talking to itself.
 * Writing them into it would assert that the Go service serves them, which is false.
 */
const handlers = toNextJsHandler(auth)

/**
 * Record that an authentication event happened, and nothing about what was in it (FR-028).
 *
 * The path and the status are enough to tell registration from sign-in from a failure from a
 * sign-out, which is what the requirement asks for. The request body is never read here, so a
 * password cannot reach a log by accident — a log that echoes what someone typed is a log that
 * leaks their credentials, and it is the same reasoning the Go service's request logger already
 * follows.
 *
 * The email is not recorded either. It would make the log a list of who has an account here, and
 * a vault is private.
 */
function withLogging(handler: (request: Request) => Promise<Response>) {
  return async (request: Request): Promise<Response> => {
    const started = Date.now()
    const action = new URL(request.url).pathname.replace(/^\/api\/auth\//, '')
    const response = await handler(request)
    console.info(
      JSON.stringify({
        event: 'auth',
        action,
        method: request.method,
        status: response.status,
        outcome: response.ok ? 'accepted' : 'refused',
        duration_ms: Date.now() - started,
      }),
    )
    return response
  }
}

export const GET = withLogging(handlers.GET)
export const POST = withLogging(handlers.POST)
