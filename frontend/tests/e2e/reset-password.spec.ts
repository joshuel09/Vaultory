import { expect, test, type Page } from '@playwright/test'
import { fillWhenReady, gotoReady } from './support'
import { latestMessageTo, linkIn } from './mailpit'

const OLD = 'a-long-enough-password'
const NEW = 'a-brand-new-password-x'

async function register(page: Page) {
  const email = `reset-${Date.now()}-${Math.random().toString(36).slice(2, 8)}@example.test`
  await gotoReady(page, '/register')
  await fillWhenReady(page.getByLabel(/^email/i), email)
  await fillWhenReady(page.getByLabel(/^password/i), OLD)
  await page.getByRole('button', { name: /create my vault/i }).click()
  await page.waitForURL('**/collection')
  /*
   * Let the registration settle before returning.
   *
   * AuthForm calls router.push then router.refresh, and the refresh refetches /collection's RSC
   * payload. waitForURL returns before that lands, so the caller's clearCookies can fall between
   * the two — and the refetch then arrives without a cookie, is redirected by the middleware, and
   * that redirect pre-empts whatever navigation the next step started. It surfaced as
   * "goto /forgot-password interrupted by another navigation to /sign-in?next=/collection",
   * which names neither the cause nor the test that caused it.
   */
  await page.waitForLoadState('networkidle')
  return email
}

async function requestReset(page: Page, email: string) {
  /*
   * Leave the vault before dropping the session.
   *
   * AuthForm calls router.push then router.refresh, and the refresh refetches /collection's RSC
   * payload. Clearing cookies while that is in flight makes it arrive unauthenticated; the
   * middleware redirects it, and the client follows that redirect as a navigation, which pre-empts
   * whatever this function does next. It surfaced as "goto /forgot-password interrupted by another
   * navigation to /sign-in?next=/collection" — an error naming neither the cause nor the step that
   * caused it.
   *
   * Navigating to the landing page first both cancels that fetch and leaves the browser somewhere
   * the middleware does not guard, so there is nothing left to redirect when the cookies go.
   * Waiting on networkidle alone was not enough: the refresh can start after the idle window.
   */
  await gotoReady(page, '/')
  await page.context().clearCookies()
  await gotoReady(page, '/forgot-password')
  await fillWhenReady(page.getByLabel(/^email/i), email)
  await page.getByRole('button', { name: /send me a link/i }).click()
  await expect(page.getByTestId('reset-requested')).toBeVisible()
}

test.describe('resetting a forgotten password', () => {
  test.beforeEach(async ({ page }) => page.context().clearCookies())

  // US2, FR-007, FR-013, FR-014, SC-001, SC-006 — walkthrough C.
  test('a forgotten password can be replaced, and the old one stops working', async ({
    page,
    request,
  }) => {
    const email = await register(page)
    await requestReset(page, email)

    const message = await latestMessageTo(request, email)
    expect(message.subject).toMatch(/reset your vaultory password/i)
    // FR-024: the lifetime, and what it does to other devices. Matched against the message with
    // its line wrapping flattened — the template wraps at a sensible width, and an assertion that
    // breaks when a sentence moves across lines is testing the layout rather than the content.
    const flat = message.text.replace(/\s+/g, ' ')
    expect(flat).toMatch(/one hour/i)
    expect(flat).toMatch(/sign out every other device/i)
    expect(flat).toMatch(/did not ask for this/i)

    await page.goto(linkIn(message.text))

    // FR-013: too short is refused without spending the link.
    await fillWhenReady(page.getByLabel(/new password/i), 'short')
    await page.getByRole('button', { name: /set my new password/i }).click()
    // Scoped to the announced error, not the page: the same words appear in the page's lead, and
    // matching either would let this pass without the refusal ever happening.
    await expect(page.getByRole('alert').filter({ hasText: /at least 12 characters/i })).toBeVisible()

    // FR-014: the link still works, and completing it lands them in their collection rather
    // than on a sign-in form — Better Auth signs nobody in, so this is ours.
    await fillWhenReady(page.getByLabel(/new password/i), NEW)
    await page.getByRole('button', { name: /set my new password/i }).click()
    await page.waitForURL('**/collection')
    await expect(page.getByTestId('sign-out')).toBeVisible()

    // FR-014a: and the address is verified, because they read mail sent to it.
    await expect(page.getByTestId('unverified-notice')).toHaveCount(0)

    // SC-006: the old password is refused, the new one works.
    // Wait for sign-out to land before navigating. Without this the session can still exist when
    // /sign-in is requested, which redirects a signed-in collector straight back to their
    // collection — so the email field never appears and the failure looks like a missing control
    // rather than a race. It passed on desktop and failed on the slower emulated devices.
    await page.getByTestId('sign-out').click()
    await page.waitForURL(/\/$/)
    await gotoReady(page, '/sign-in')
    await fillWhenReady(page.getByLabel(/^email/i), email)
    await fillWhenReady(page.getByLabel(/^password/i), OLD)
    await page.getByRole('button', { name: /^sign in$/i }).click()
    await expect(page.getByTestId('form-error')).toBeVisible()

    await fillWhenReady(page.getByLabel(/^password/i), NEW)
    await page.getByRole('button', { name: /^sign in$/i }).click()
    await page.waitForURL('**/collection')
  })

  /**
   * FR-012 — Better Auth does not do this. `request-password-reset` inserts a row and deletes
   * nothing, so without the invalidation in lib/auth.ts three working links could exist at once.
   * A stale verification link is harmless; a stale reset link is a way into a vault.
   */
  test('asking again makes the previous link stop working', async ({ page, request }) => {
    const email = await register(page)

    await requestReset(page, email)
    const first = linkIn((await latestMessageTo(request, email)).text)

    await requestReset(page, email)
    const second = linkIn((await latestMessageTo(request, email)).text)
    expect(second).not.toBe(first)

    // Refused before a password is even asked for: Better Auth's callback rejects the token and
    // redirects with an error, so the form is never offered. Better than accepting a password and
    // then failing.
    await page.goto(first)
    await expect(page.getByTestId('reset-link-invalid')).toBeVisible()
    await expect(page.getByLabel(/new password/i)).toHaveCount(0)

    // And the newest one still works.
    await page.goto(second)
    await fillWhenReady(page.getByLabel(/new password/i), NEW)
    await page.getByRole('button', { name: /set my new password/i }).click()
    await page.waitForURL('**/collection')
  })

  // FR-011 — a spent link cannot be spent again.
  test('a link cannot be used twice', async ({ page, request }) => {
    const email = await register(page)
    await requestReset(page, email)
    const link = linkIn((await latestMessageTo(request, email)).text)

    await page.goto(link)
    await fillWhenReady(page.getByLabel(/new password/i), NEW)
    await page.getByRole('button', { name: /set my new password/i }).click()
    await page.waitForURL('**/collection')

    await page.context().clearCookies()
    await page.goto(link)
    await expect(page.getByTestId('reset-link-invalid')).toBeVisible()
    await expect(page.getByLabel(/new password/i)).toHaveCount(0)
  })

  // FR-008, SC-003 — the form must not reveal who has an account.
  test('asking for a reset reveals nothing about the address', async ({ page }) => {
    const known = await register(page)
    await page.context().clearCookies()

    const seen: string[] = []
    for (const address of [known, `nobody-${Date.now()}@example.test`]) {
      await gotoReady(page, '/forgot-password')
      await fillWhenReady(page.getByLabel(/^email/i), address)
      await page.getByRole('button', { name: /send me a link/i }).click()
      await expect(page.getByTestId('reset-requested')).toBeVisible()
      seen.push((await page.getByTestId('reset-requested').innerText()).trim())
    }
    expect(seen[0]).toBe(seen[1])
  })
})

/*
 * Issue #41 — the reset succeeded, the sign-in after it did not, and the collector was told
 * nothing.
 *
 * Driven by intercepting the sign-in call rather than by exhausting the real rate limiter. The
 * limiter is per address with a fifteen-minute window and is shared by every test in the run, so
 * provoking it for real would be slow, would leak into unrelated tests, and would still only
 * reproduce one of the several ways this request can fail. What is under test is what the product
 * does when it fails, which is the same either way.
 *
 * The cause was confirmed against the running stack separately: setting the sign-in limiter over
 * its ceiling and running the journey above reproduces exactly the observed failure, landing on
 * /sign-in?next=/collection with nothing said.
 */
test.describe('when the sign-in after a reset is refused', () => {
  test('the collector is told the password was changed, rather than silently bounced', async ({
    page,
    request,
  }) => {
    const email = await register(page)
    await requestReset(page, email)
    const message = await latestMessageTo(request, email)
    await page.goto(linkIn(message.text))

    // Refuse the sign-in that follows the reset, the way the rate limiter does. The reset itself
    // is left alone: it must still succeed, because that is what makes the silence damaging.
    await page.route('**/api/auth/sign-in/email', (route) =>
      route.fulfill({
        status: 429,
        contentType: 'application/json',
        body: JSON.stringify({ message: 'Too many requests' }),
      }),
    )

    await fillWhenReady(page.getByLabel(/new password/i), NEW)
    await page.getByRole('button', { name: /set my new password/i }).click()

    await page.waitForURL(/\/sign-in/)
    const notice = page.getByTestId('reset-complete')
    await expect(notice).toBeVisible()
    // The part that matters: they must not conclude the reset failed and go back to a spent link.
    await expect(notice).toContainText(/password has been changed/i)
    await expect(notice).toContainText(/wait a few minutes/i)

    // And it really did change: the new password works once the refusal is lifted.
    await page.unroute('**/api/auth/sign-in/email')
    await fillWhenReady(page.getByLabel(/^email/i), email)
    await fillWhenReady(page.getByLabel(/^password/i), NEW)
    await page.getByRole('button', { name: /sign in/i }).click()
    await page.waitForURL('**/collection')
    await expect(page.getByTestId('sign-out')).toBeVisible()
  })
})
