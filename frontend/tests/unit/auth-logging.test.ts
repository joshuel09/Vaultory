import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { actionOf } from '@/lib/auth-logging'

const source = readFileSync(resolve(__dirname, '../../app/api/auth/[...all]/route.ts'), 'utf8')

/**
 * FR-016, FR-021, SC-007.
 *
 * Recovery events must be recorded, and their contents must not be. This logger was doing the
 * second thing wrong: `/api/auth/reset-password/:token` put the token straight into the log, and a
 * check against the running stack found 45 of them.
 */
describe('the authentication logger', () => {
  it('records the event, never the request body', () => {
    expect(source).toMatch(/event:\s*'auth'/)
    expect(source).toMatch(/status:\s*response\.status/)
    // Reading the body is how a password reaches a log by accident.
    expect(source).not.toMatch(/request\.(?:json|text|formData)\(/)
  })

  it('drops dynamic path segments, so a token cannot become an action', () => {
    expect(actionOf('/api/auth/reset-password/RWDXKO80ywIDmFMemzr2pXCd')).toBe('reset-password')
    expect(actionOf('/api/auth/sign-in/email')).toBe('sign-in/email')
    expect(actionOf('/api/auth/verify-email')).toBe('verify-email')
    expect(actionOf('/api/auth/request-password-reset')).toBe('request-password-reset')
    // A whitelist, so the next dynamic segment somebody adds is dropped without being remembered.
    expect(actionOf('/api/auth/something/AbC123XyZ456')).toBe('something')
    expect(actionOf('/api/auth/callback/9f3b2a1c8d7e')).toBe('callback')
  })

  it('keeps the filter a whitelist rather than a list of things to redact', () => {
    const filter = readFileSync(resolve(__dirname, '../../lib/auth-logging.ts'), 'utf8')
    expect(filter).toMatch(/\/\^\[a-z\]\[a-z0-9-\]\*\$\//)
  })
})
