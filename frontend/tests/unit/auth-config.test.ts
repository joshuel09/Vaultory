import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const source = readFileSync(resolve(__dirname, '../../lib/auth.ts'), 'utf8')

/**
 * FR-005, and FR-010's sliding half.
 *
 * Research Decision 5 chose Better Auth's default hasher deliberately: scrypt is memory-hard and
 * meets "deliberately slow, salted, not recoverable". A default that nothing checks is one a later
 * configuration change can weaken without anyone noticing, so this asserts the decision rather
 * than trusting it to survive.
 */
describe('the authentication configuration', () => {
  it('does not override the password hasher', () => {
    expect(source).not.toMatch(/password\s*:\s*\{/)
    expect(source).not.toMatch(/\bhash\s*:/)
    expect(source).not.toMatch(/\bverify\s*:/)
  })

  it('requires at least 12 characters', () => {
    expect(source).toMatch(/minPasswordLength:\s*12/)
  })

  it('enables email and password, and no social provider', () => {
    expect(source).toMatch(/emailAndPassword:\s*\{[^}]*enabled:\s*true/s)
    expect(source).not.toMatch(/socialProviders/)
  })

  it('slides the session window and stores rate limits durably', () => {
    // 30 days, extended on use (FR-010). The 90-day ceiling is Go's, by design.
    expect(source).toMatch(/expiresIn:\s*60 \* 60 \* 24 \* 30/)
    expect(source).toMatch(/updateAge:/)
    // In-memory counters reset on restart, which would make FR-027 evaporate on every deploy.
    expect(source).toMatch(/storage:\s*'database'/)
  })

  it('never hard-codes a secret', () => {
    expect(source).toMatch(/process\.env\.BETTER_AUTH_SECRET/)
    expect(source).not.toMatch(/secret:\s*['"][^'"]{8,}['"]/)
  })
})
