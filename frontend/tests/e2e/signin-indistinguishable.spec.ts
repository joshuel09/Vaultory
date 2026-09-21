import { expect, test } from '@playwright/test'

/**
 * FR-008, SC-006 — a wrong password and an unregistered email must be indistinguishable.
 *
 * Deliberately not a quickstart walkthrough: eyeballing two responses is not evidence. If the two
 * differ in status, body, or noticeably in time, sign-in becomes a way to discover who has an
 * account here, which for a private collection is a disclosure in itself.
 *
 * This lives in the browser suite rather than in Go's contract tests, where tasks.md placed it.
 * Sign-in is served by Next.js; the Go service never sees an attempt, so a Go test could not
 * reach the endpoint at all.
 */
test.describe('sign-in tells an attacker nothing', () => {
  const password = 'a-long-enough-password'

  async function attempt(request: import('@playwright/test').APIRequestContext, email: string) {
    const origin = process.env.VAULTORY_BASE_URL ?? 'http://localhost:3000'
    const started = Date.now()
    const response = await request.post('/api/auth/sign-in/email', {
      headers: { Origin: origin },
      data: { email, password: 'wrong-password-entirely' },
      failOnStatusCode: false,
    })
    const body = await response.text()
    return { status: response.status(), body, ms: Date.now() - started }
  }

  test('a wrong password and an unknown email answer the same way', async ({ request }) => {
    // One address that exists, one that never will.
    const registered = `known-${Date.now()}@example.test`
    const origin = process.env.VAULTORY_BASE_URL ?? 'http://localhost:3000'
    const created = await request.post('/api/auth/sign-up/email', {
      headers: { Origin: origin },
      data: { email: registered, password, name: 'known' },
    })
    expect(created.ok(), 'the fixture account must be created').toBeTruthy()

    const unknown = `never-${Date.now()}-${Math.random().toString(36).slice(2)}@example.test`

    // Several rounds: a single pair can coincide, and the sign-in limit is 10 per account.
    const rounds = 4
    const known: number[] = []
    const absent: number[] = []

    for (let i = 0; i < rounds; i++) {
      const a = await attempt(request, registered)
      const b = await attempt(request, unknown)

      expect(a.status, 'status must not reveal whether the account exists').toBe(b.status)
      expect(a.body, 'body must not reveal whether the account exists').toBe(b.body)
      known.push(a.ms)
      absent.push(b.ms)
    }

    // Timing. Not a strict constant-time claim — this is a network round trip through a dev
    // server — but a wrong password should not take dramatically longer than a missing account,
    // which is what happens if one path hashes and the other returns early.
    const mean = (xs: number[]) => xs.reduce((t, x) => t + x, 0) / xs.length
    const ratio = mean(known) / mean(absent)
    expect(
      ratio,
      `wrong password averaged ${mean(known).toFixed(0)}ms against ${mean(absent).toFixed(0)}ms ` +
        'for an unknown address; a large gap lets sign-in be used to enumerate accounts',
    ).toBeLessThan(5)
    expect(ratio).toBeGreaterThan(0.2)
  })
})
