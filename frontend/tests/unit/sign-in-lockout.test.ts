import { describe, expect, it } from 'vitest'
import { APIError, isAPIError } from 'better-auth/api'
import {
  addressKey,
  clearAddress,
  LOCKED_CODE,
  LOCKED_MESSAGE,
  recordOutcome,
  refuseIfLocked,
  type Queryable,
} from '@/lib/sign-in-lockout'

/**
 * Feature 013, T006. The rule is in PostgreSQL and tested there by the Go integration suite; this
 * covers the part that is not — which of Better Auth's answers counts as a failure, and the key.
 */

const secret = 'a-secret-of-at-least-thirty-two-characters'

/** A stand-in for the pool: records each call, and answers sign_in_is_locked as told. */
function fakeDb(locked = false): Queryable & { calls: string[] } {
  const calls: string[] = []
  return {
    calls,
    query: async (text: string) => {
      calls.push(text.replace(/^SELECT (\w+).*$/, '$1'))
      return { rows: [{ locked }] }
    },
  }
}

describe('the address key', () => {
  it('is the same address however it is typed', () => {
    expect(addressKey('  Aerith@Example.TEST ', secret)).toBe(addressKey('aerith@example.test', secret))
  })

  it('differs between addresses and between secrets', () => {
    expect(addressKey('a@example.test', secret)).not.toBe(addressKey('b@example.test', secret))
    expect(addressKey('a@example.test', secret)).not.toBe(addressKey('a@example.test', `${secret}x`))
  })

  it('does not contain the address', () => {
    const key = addressKey('aerith@example.test', secret)
    expect(key).not.toContain('aerith')
    expect(key).toMatch(/^[A-Za-z0-9_-]{43}$/)
  })
})

describe('before sign-in', () => {
  it('refuses a locked address with 429 and the one message', async () => {
    const err = await refuseIfLocked(fakeDb(true), secret, 'a@example.test').catch((e) => e)
    expect(isAPIError(err)).toBe(true)
    expect(err.statusCode).toBe(429)
    expect(err.body.code).toBe(LOCKED_CODE)
    expect(err.body.message).toBe(LOCKED_MESSAGE)
    // FR-007: says what happened and the way back, not whether there is an account.
    expect(LOCKED_MESSAGE).toMatch(/too many/i)
    expect(LOCKED_MESSAGE).toMatch(/reset your password/i)
    expect(LOCKED_MESSAGE).not.toMatch(/account/i)
  })

  it('lets an address that is not locked through', async () => {
    await expect(refuseIfLocked(fakeDb(false), secret, 'a@example.test')).resolves.toBeUndefined()
  })

  it('asks nothing when there is no address to ask about', async () => {
    const db = fakeDb(true)
    await refuseIfLocked(db, secret, undefined)
    expect(db.calls).toEqual([])
  })
})

describe('after sign-in', () => {
  const wrong = new APIError('UNAUTHORIZED', {
    message: 'Invalid email or password',
    code: 'INVALID_EMAIL_OR_PASSWORD',
  })

  it('records a wrong password or an unknown address as a failure', async () => {
    const db = fakeDb()
    await recordOutcome(db, secret, 'a@example.test', wrong)
    expect(db.calls).toEqual(['sign_in_record_failure'])
  })

  it('clears the count on a successful sign-in', async () => {
    const db = fakeDb()
    await recordOutcome(db, secret, 'a@example.test', { token: 't', user: { id: 'u' } })
    expect(db.calls).toEqual(['sign_in_clear'])
  })

  it('does not count an unverified address (FR-010) or a refusal', async () => {
    const db = fakeDb()
    await recordOutcome(db, secret, 'a@example.test',
      new APIError('FORBIDDEN', { message: 'Email not verified', code: 'EMAIL_NOT_VERIFIED' }))
    await recordOutcome(db, secret, 'a@example.test',
      new APIError('TOO_MANY_REQUESTS', { message: LOCKED_MESSAGE, code: LOCKED_CODE }))
    expect(db.calls).toEqual([])
  })
})

describe('a completed password reset', () => {
  it('clears the address (FR-005)', async () => {
    const db = fakeDb()
    await clearAddress(db, secret, 'A@Example.test')
    expect(db.calls).toEqual(['sign_in_clear'])
  })
})
