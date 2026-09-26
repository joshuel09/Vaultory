import { expect, test, type Page } from '@playwright/test'
import { gotoReady } from './support'
import { latestMessageTo, linkIn } from './mailpit'

const OLD = 'a-long-enough-password'
const NEW = 'a-brand-new-password-x'

async function register(page: Page) {
  const email = `reset-${Date.now()}-${Math.random().toString(36).slice(2, 8)}@example.test`
  await gotoReady(page, '/register')
  await page.getByLabel(/^email/i).fill(email)
  await page.getByLabel(/^password/i).fill(OLD)
  await page.getByRole('button', { name: /create my vault/i }).click()
  await page.waitForURL('**/collection')
  return email
}

async function requestReset(page: Page, email: string) {
  await page.context().clearCookies()
  await gotoReady(page, '/forgot-password')
  await page.getByLabel(/^email/i).fill(email)
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
    await page.getByLabel(/new password/i).fill('short')
    await page.getByRole('button', { name: /set my new password/i }).click()
    // Scoped to the announced error, not the page: the same words appear in the page's lead, and
    // matching either would let this pass without the refusal ever happening.
    await expect(page.getByRole('alert').filter({ hasText: /at least 12 characters/i })).toBeVisible()

    // FR-014: the link still works, and completing it lands them in their collection rather
    // than on a sign-in form — Better Auth signs nobody in, so this is ours.
    await page.getByLabel(/new password/i).fill(NEW)
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
    await page.getByLabel(/^email/i).fill(email)
    await page.getByLabel(/^password/i).fill(OLD)
    await page.getByRole('button', { name: /^sign in$/i }).click()
    await expect(page.getByTestId('form-error')).toBeVisible()

    await page.getByLabel(/^password/i).fill(NEW)
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
    await page.getByLabel(/new password/i).fill(NEW)
    await page.getByRole('button', { name: /set my new password/i }).click()
    await page.waitForURL('**/collection')
  })

  // FR-011 — a spent link cannot be spent again.
  test('a link cannot be used twice', async ({ page, request }) => {
    const email = await register(page)
    await requestReset(page, email)
    const link = linkIn((await latestMessageTo(request, email)).text)

    await page.goto(link)
    await page.getByLabel(/new password/i).fill(NEW)
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
      await page.getByLabel(/^email/i).fill(address)
      await page.getByRole('button', { name: /send me a link/i }).click()
      await expect(page.getByTestId('reset-requested')).toBeVisible()
      seen.push((await page.getByTestId('reset-requested').innerText()).trim())
    }
    expect(seen[0]).toBe(seen[1])
  })
})
