import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const source = readFileSync(resolve(__dirname, '../../lib/auth.ts'), 'utf8')

/** Drop comments, so an assertion about configuration is not satisfied or broken by prose. */
function stripComments(code: string): string {
  return code.replace(/\/\*[\s\S]*?\*\//g, '').replace(/\/\/[^\n]*/g, '')
}

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

  // FR-027 is the per-account sign-in rule. The global ceiling is abuse protection and must stay
  // well clear of it, or ordinary use behind a shared address trips the wrong limit first.
  it('limits sign-in attempts per account, without throttling ordinary use', () => {
    // The default is ten; the value is configurable so the browser suite, which signs in dozens
    // of times from one address, is not throttled by a limit meant for one person.
    expect(source).toMatch(/'\/sign-in\/email':\s*\{[\s\S]*?BETTER_AUTH_SIGNIN_MAX \?\? 10/)
    const globalMax = Number(/\n\s*max:\s*(\d+),/.exec(source)?.[1] ?? 0)
    expect(globalMax).toBeGreaterThan(500)
  })

  /*
   * FR-002a, and a test guarding an absence.
   *
   * autoSignInAfterVerification is opt-in, so leaving it out is correct — and "correct by
   * omission" is the kind of thing a later edit undoes without anyone noticing, because adding it
   * looks like an improvement. A verification link lives 24 hours; treating it as a way in would
   * make a forwarded or archived message access to a vault for a day.
   */
  it('does not sign anyone in after verification', () => {
    // Look for an assignment, not a mention: the configuration explains at length why this option
    // is left out, and a blunter check would fail on its own comment.
    const assigned = /autoSignInAfterVerification\s*:/.test(stripComments(source))
    expect(assigned, 'autoSignInAfterVerification must not be set — FR-002a').toBe(false)
  })

  it('sends a verification message on sign-up, valid for a day', () => {
    expect(source).toMatch(/sendOnSignUp:\s*true/)
    expect(source).toMatch(/expiresIn:\s*60 \* 60 \* 24\b/)
    expect(source).toMatch(/sendVerificationEmail:/)
  })

  /*
   * FR-015, and the single most consequential line in this feature.
   *
   * Better Auth stores reset tokens plain by default, which makes a copy of the verification
   * table a set of working reset links to every account with an outstanding reset.
   */
  it('stores recovery tokens hashed, never plain', () => {
    expect(source).toMatch(/storeIdentifier:\s*'hashed'/)
    expect(source).not.toMatch(/storeIdentifier:\s*'plain'/)
  })

  it('never hard-codes a secret', () => {
    expect(source).toMatch(/process\.env\.BETTER_AUTH_SECRET/)
    expect(source).not.toMatch(/secret:\s*['"][^'"]{8,}['"]/)
  })
})
