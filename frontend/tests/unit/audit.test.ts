import { afterEach, describe, expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { APIError } from 'better-auth/api'
import { recordAccountEvent, signInEvent } from '@/lib/audit'
import type { Queryable } from '@/lib/sign-in-lockout'

/**
 * Feature 014, T010 — account events in the audit log. The table, its append-only enforcement and
 * the collection events are tested against PostgreSQL by the Go integration suite; this covers
 * which of Better Auth's answers become which entry, and that a failed write never refuses.
 */

const key = 'one-way-address-key'

function recorder(): Queryable & { inserts: unknown[][] } {
  const inserts: unknown[][] = []
  return {
    inserts,
    query: async (_text: string, values: unknown[]) => {
      inserts.push(values)
      return { rows: [] }
    },
  }
}

describe('a sign-in attempt in the log', () => {
  it('a wrong password or an unknown address is one failure, carrying the key, not the address', () => {
    const wrong = new APIError('UNAUTHORIZED', {
      message: 'Invalid email or password',
      code: 'INVALID_EMAIL_OR_PASSWORD',
    })
    expect(signInEvent(wrong, key)).toEqual({
      action: 'account.sign_in_failed',
      outcome: 'failed',
      subjectId: key,
    })
  })

  it('a success names the account', () => {
    expect(signInEvent({ token: 't', user: { id: 'user-1' } }, key)).toEqual({
      action: 'account.signed_in',
      outcome: 'succeeded',
      accountId: 'user-1',
      subjectId: key,
    })
  })

  it('an unverified address is not an event', () => {
    const unverified = new APIError('FORBIDDEN', { message: 'Email not verified', code: 'EMAIL_NOT_VERIFIED' })
    expect(signInEvent(unverified, key)).toBeNull()
  })
})

describe('writing an entry', () => {
  afterEach(() => vi.restoreAllMocks())

  it('inserts the action, outcome, account and subject — and nothing else', async () => {
    const db = recorder()
    await recordAccountEvent(db, { action: 'account.registered', outcome: 'succeeded', accountId: 'user-1' })
    expect(db.inserts).toEqual([['account.registered', 'succeeded', 'user-1', null]])
  })

  it('never throws when the write fails, and reports it without the request (FR-005)', async () => {
    const logged = vi.spyOn(console, 'error').mockImplementation(() => {})
    const failing: Queryable = {
      query: async () => {
        throw Object.assign(new Error('relation "audit_events" does not exist'), { code: '42P01' })
      },
    }
    await expect(
      recordAccountEvent(failing, { action: 'account.signed_in', outcome: 'succeeded', accountId: 'user-1' }),
    ).resolves.toBeUndefined()
    expect(logged).toHaveBeenCalledTimes(1)
    const line = JSON.parse(String(logged.mock.calls[0]?.[0]))
    expect(line).toEqual({ event: 'audit_write_failed', action: 'account.signed_in', code: '42P01' })
  })
})

describe('the wiring in lib/auth.ts', () => {
  const source = readFileSync(resolve(__dirname, '../../lib/auth.ts'), 'utf8')

  it('records every account event the spec lists', () => {
    for (const action of [
      'account.registered',
      'account.sign_in_refused',
      'account.signed_out',
      'account.email_verified',
      'account.password_reset',
    ]) {
      expect(source).toContain(`'${action}'`)
    }
    expect(source).toMatch(/signInEvent\(ctx\.context\.returned/)
  })

  it('counts only a sign-out as a sign-out, and only verification as verification', () => {
    expect(source).toMatch(/context\?\.path !== '\/sign-out'/)
    expect(source).toMatch(/context\?\.path !== '\/verify-email'/)
  })
})
