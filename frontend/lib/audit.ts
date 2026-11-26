import { isAPIError } from 'better-auth/api'
import type { Queryable } from '@/lib/sign-in-lockout'

/**
 * Account events in the audit log (feature 014, issue #100).
 *
 * The Go service records collection changes; this records what happens to accounts, from Better
 * Auth's hooks in lib/auth.ts. Each entry is the action, its outcome, the account when there is
 * one, and for sign-in attempts feature 013's one-way address key — never the address, a password,
 * a token or a network address (FR-003).
 *
 * The table refuses changes and early deletion itself (migration 000014); the frontend's role may
 * only add to it.
 */

export type AccountAction =
  | 'account.registered'
  | 'account.signed_in'
  | 'account.sign_in_failed'
  | 'account.sign_in_refused'
  | 'account.signed_out'
  | 'account.email_verified'
  | 'account.password_reset'

export type AccountEvent = {
  action: AccountAction
  outcome: 'succeeded' | 'failed' | 'refused'
  accountId?: string | null
  subjectId?: string | null
}

/**
 * Add one entry. Never throws: a log that cannot be written must not become a way to keep every
 * collector out (FR-005). The failure is reported to the operator instead — with the action and the
 * database's error code, and nothing from the request.
 */
export async function recordAccountEvent(db: Queryable, event: AccountEvent): Promise<void> {
  try {
    await db.query(
      'INSERT INTO audit_events (action, outcome, account_id, subject_id) VALUES ($1, $2, $3, $4)',
      [event.action, event.outcome, event.accountId ?? null, event.subjectId ?? null],
    )
  } catch (error) {
    const code = (error as { code?: unknown })?.code
    console.error(
      JSON.stringify({
        event: 'audit_write_failed',
        action: event.action,
        code: typeof code === 'string' ? code : 'unknown',
      }),
    )
  }
}

/**
 * What a sign-in attempt's answer means for the log, or null when it is not an event: a wrong
 * password and an unknown address are the same failure, as they are to the collector (feature 013,
 * FR-006); an unverified address is neither a success nor a password failure.
 */
export function signInEvent(returned: unknown, addressKey: string): AccountEvent | null {
  if (isAPIError(returned)) {
    if (returned.statusCode === 401 && returned.body?.code === 'INVALID_EMAIL_OR_PASSWORD') {
      return { action: 'account.sign_in_failed', outcome: 'failed', subjectId: addressKey }
    }
    return null
  }
  const accountId = (returned as { user?: { id?: unknown } } | null)?.user?.id
  if (typeof accountId !== 'string') return null
  return { action: 'account.signed_in', outcome: 'succeeded', accountId, subjectId: addressKey }
}
