import { createHmac } from 'node:crypto'
import { APIError, isAPIError } from 'better-auth/api'

/**
 * Sign-in is locked for an email address after five consecutive failures, for fifteen minutes
 * (feature 013, issue #99).
 *
 * The rule itself is in PostgreSQL — migration 000013's three functions — so every instance shares
 * one count and a burst of failures cannot race past it. This module only decides, from what Better
 * Auth returned, which of them to call, and computes the key they are called with.
 *
 * Counted per address whether or not it has an account. Counting only real accounts would make the
 * lock itself the way to learn which addresses are registered (FR-006).
 */

/** Five and fifteen: the company security guideline, and feature 004's lock duration. */
export const MAX_FAILURES = 5
export const LOCK_MINUTES = 15

/** The one message a locked address gets, whether or not it has an account (FR-006, FR-007). */
export const LOCKED_MESSAGE = `Too many sign-in attempts. Try again in ${LOCK_MINUTES} minutes, or reset your password.`
export const LOCKED_CODE = 'SIGN_IN_LOCKED'

/** What the functions are called on: the pool, or a stand-in in tests. */
export type Queryable = { query: (text: string, values: unknown[]) => Promise<unknown> }

/**
 * The address as stored: trimmed, lower-cased, then an HMAC with the server's secret (FR-009).
 *
 * Not a plain hash: email addresses are not secret, so anyone holding a table of plain hashes could
 * test candidates against it. With the secret as the key, they would need the secret as well.
 */
export function addressKey(email: string, secret: string): string {
  return createHmac('sha256', secret).update(email.trim().toLowerCase(), 'utf8').digest('base64url')
}

async function isLocked(db: Queryable, key: string): Promise<boolean> {
  const result = (await db.query('SELECT sign_in_is_locked($1) AS locked', [key])) as {
    rows: { locked: boolean }[]
  }
  return result.rows[0]?.locked === true
}

/**
 * Before the password is checked: refuse a locked address. Refusing here, rather than after the
 * check, means a right password during the lock looks exactly like a wrong one, and that an attempt
 * during the lock neither counts nor extends it.
 */
export async function refuseIfLocked(db: Queryable, secret: string, email: unknown): Promise<void> {
  if (typeof email !== 'string' || email === '') return
  if (await isLocked(db, addressKey(email, secret))) {
    throw new APIError('TOO_MANY_REQUESTS', { message: LOCKED_MESSAGE, code: LOCKED_CODE })
  }
}

/**
 * After Better Auth has answered: a wrong password or an unknown address — the same 401 for both —
 * is a failure; a sign-in is a success. Anything else is neither: an unverified address (403) is
 * not a password failure (FR-010), and a refusal (429) was never checked.
 */
export async function recordOutcome(
  db: Queryable,
  secret: string,
  email: unknown,
  returned: unknown,
): Promise<void> {
  if (typeof email !== 'string' || email === '') return
  const key = addressKey(email, secret)
  if (isAPIError(returned)) {
    if (returned.statusCode === 401 && returned.body?.code === 'INVALID_EMAIL_OR_PASSWORD') {
      await db.query('SELECT sign_in_record_failure($1)', [key])
    }
    return
  }
  await db.query('SELECT sign_in_clear($1)', [key])
}

/** A completed password reset ends any lock on the account's address (FR-005). */
export async function clearAddress(db: Queryable, secret: string, email: string): Promise<void> {
  await db.query('SELECT sign_in_clear($1)', [addressKey(email, secret)])
}
