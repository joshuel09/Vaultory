import { expect, test, type APIRequestContext } from '@playwright/test'
import { latestMessageTo, linkIn } from './mailpit'

/**
 * Feature 013 — sign-in is locked for an email address after five consecutive failures, for
 * fifteen minutes, whether or not the address has an account.
 *
 * Driven through the endpoint rather than the form, like signin-indistinguishable.spec.ts: what is
 * under test is the exact status and body an attacker would see, and the form only displays the
 * message. Every address here is fresh, so the per-network ceiling raised for the suite is never
 * what refuses.
 */
const origin = process.env.VAULTORY_BASE_URL ?? 'http://localhost:3000'
const PASSWORD = 'a-long-enough-password'
const NEW_PASSWORD = 'a-brand-new-password-y'

function freshAddress(label: string) {
  return `${label}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}@example.test`
}

async function register(request: APIRequestContext, email: string) {
  const created = await request.post('/api/auth/sign-up/email', {
    headers: { Origin: origin },
    data: { email, password: PASSWORD, name: 'lockout' },
  })
  expect(created.ok(), 'the fixture account must be created').toBeTruthy()
}

async function signIn(request: APIRequestContext, email: string, password: string) {
  const response = await request.post('/api/auth/sign-in/email', {
    headers: { Origin: origin },
    data: { email, password },
    failOnStatusCode: false,
  })
  return { status: response.status(), body: await response.text() }
}

test.describe('the sign-in lock', () => {
  // T007 — FR-002, FR-006, SC-001, SC-002.
  test('locks after five failures, the same way for an address with and without an account', async ({
    request,
  }) => {
    const registered = freshAddress('locked-known')
    await register(request, registered)
    const unknown = freshAddress('locked-never')

    const known: { status: number; body: string }[] = []
    const absent: { status: number; body: string }[] = []
    for (let i = 0; i < 6; i++) {
      known.push(await signIn(request, registered, 'wrong-password-entirely'))
      absent.push(await signIn(request, unknown, 'wrong-password-entirely'))
    }

    expect(known.map((r) => r.status)).toEqual([401, 401, 401, 401, 401, 429])
    // Status and body, attempt by attempt: nothing distinguishes the address with an account.
    expect(known, 'the responses must not reveal whether the account exists').toEqual(absent)
    const lockedResponse = known.at(-1)
    expect(lockedResponse?.body).toContain('SIGN_IN_LOCKED')
    expect(lockedResponse?.body).toMatch(/too many sign-in attempts/i)

    // The right password does not get past the lock, and is refused exactly as a wrong one is.
    const right = await signIn(request, registered, PASSWORD)
    expect(right).toEqual(lockedResponse)
  })

  // T010 — FR-005, SC-003, User Story 3.
  test('a completed password reset ends the lock; only requesting one does not', async ({ request }) => {
    const email = freshAddress('locked-reset')
    await register(request, email)
    for (let i = 0; i < 5; i++) await signIn(request, email, 'wrong-password-entirely')
    expect((await signIn(request, email, PASSWORD)).status).toBe(429)

    const requested = await request.post('/api/auth/request-password-reset', {
      headers: { Origin: origin },
      data: { email, redirectTo: '/reset-password' },
    })
    expect(requested.ok()).toBeTruthy()
    // Requesting is not completing: still locked.
    expect((await signIn(request, email, PASSWORD)).status).toBe(429)

    const link = linkIn((await latestMessageTo(request, email)).text)
    const token = new URL(link).pathname.split('/').pop() ?? ''
    const reset = await request.post('/api/auth/reset-password', {
      headers: { Origin: origin },
      data: { token, newPassword: NEW_PASSWORD },
    })
    expect(reset.ok(), 'the reset must complete').toBeTruthy()

    const after = await signIn(request, email, NEW_PASSWORD)
    expect(after.status, `sign-in after the reset answered ${after.body}`).toBe(200)
  })
})
