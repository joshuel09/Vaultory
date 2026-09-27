import { expect, test } from '@playwright/test'
import { gotoReady } from './support'
import { latestMessageTo, linkIn } from './mailpit'

async function register(page: import('@playwright/test').Page) {
  const email = `verify-${Date.now()}-${Math.random().toString(36).slice(2, 8)}@example.test`
  await gotoReady(page, '/register')
  await page.getByLabel(/^email/i).fill(email)
  await page.getByLabel(/^password/i).fill('a-long-enough-password')
  await page.getByRole('button', { name: /create my vault/i }).click()
  await page.waitForURL('**/collection')
  return email
}

test.describe('verifying an email address', () => {
  test.beforeEach(async ({ page }) => page.context().clearCookies())

  // US1, FR-001, FR-005, SC-002 — walkthrough A.
  test('registering sends a message, and the vault says the address is unverified', async ({
    page,
    request,
  }) => {
    const email = await register(page)

    await expect(page.getByTestId('unverified-notice')).toBeVisible()
    await expect(page.getByTestId('resend-verification')).toBeVisible()

    const message = await latestMessageTo(request, email)
    expect(message.subject).toMatch(/verify your email/i)
    // FR-024: what it is for, how long it lasts, and what to do if unexpected.
    expect(message.text).toMatch(/24 hours/i)
    expect(message.text).toMatch(/did not create/i)
    expect(message.text).toContain(email)
  })

  // FR-002 — following the link verifies, and the notice goes away.
  test('following the link verifies the address', async ({ page, request }) => {
    const email = await register(page)
    const link = linkIn((await latestMessageTo(request, email)).text)

    await page.goto(link)
    await expect(page.getByTestId('verify-succeeded')).toBeVisible()

    await gotoReady(page, '/collection')
    await expect(page.getByTestId('unverified-notice')).toHaveCount(0)
  })

  /**
   * FR-002a, SC-013 — walkthrough B, and the one most easily broken by a later convenience.
   *
   * A verification link lives 24 hours. If following one granted a session, a forwarded or
   * archived message would be a way into somebody's vault for a day. Proving you can read an
   * inbox is not proving you know a password.
   */
  test('following the link grants no session', async ({ page, request, browser }) => {
    const email = await register(page)
    const link = linkIn((await latestMessageTo(request, email)).text)

    // A browser that has never seen this account, the way a phone would be. baseURL is passed
    // explicitly: a context made this way does not inherit the project's, so a relative
    // navigation would have nowhere to go.
    const fresh = await browser.newContext({
      baseURL: process.env.VAULTORY_BASE_URL ?? 'http://localhost:3000',
    })
    const strangerPage = await fresh.newPage()
    await strangerPage.goto(link)

    await expect(strangerPage.getByTestId('verify-succeeded')).toBeVisible()
    expect(
      await fresh.cookies(),
      'verifying an address must not hand out a session',
    ).toEqual(expect.not.arrayContaining([expect.objectContaining({ name: 'better-auth.session_token' })]))

    // And no collection content is reachable from there.
    await strangerPage.goto('/collection')
    await strangerPage.waitForURL(/\/sign-in/)
    await fresh.close()
  })
})
