import { expect, test } from '@playwright/test'
import { gotoReady } from './support'

/**
 * FR-023, FR-025, and the gap a real person found: after registering there was no way back out of
 * a vault, the landing page still offered "Sign in" to someone already signed in, and visiting
 * the sign-in page again presented a form rather than their collection.
 */
async function register(page: import('@playwright/test').Page) {
  const email = `nav-${Date.now()}-${Math.random().toString(36).slice(2, 8)}@example.test`
  await gotoReady(page, '/register')
  await page.getByLabel(/^email/i).fill(email)
  await page.getByLabel(/^password/i).fill('a-long-enough-password')
  await page.getByRole('button', { name: /create my vault/i }).click()
  await page.waitForURL('**/collection')
  return email
}

test.describe('navigation once signed in', () => {
  test.beforeEach(async ({ page }) => page.context().clearCookies())

  // FR-025: a way back out, and a way to tell which account you are using.
  test('every vault page has a header with the account and a way out', async ({ page }) => {
    const email = await register(page)

    await expect(page.getByTestId('signed-in-as')).toHaveText(email)
    await expect(page.getByTestId('sign-out')).toBeVisible()

    await gotoReady(page, '/collection/new')
    await expect(page.getByTestId('sign-out')).toBeVisible()

    // And the way home actually goes home.
    await page.getByRole('link', { name: /^vaultory$/i }).click()
    await page.waitForURL(new RegExp(`${page.url().split('/').slice(0, 3).join('/')}/?$`))
  })

  test('the landing page offers the vault, not a sign-in link', async ({ page }) => {
    await register(page)
    await gotoReady(page, '/')

    await expect(page.getByRole('link', { name: /my vault/i }).first()).toBeVisible()
    await expect(page.getByRole('link', { name: /^sign in$/i })).toHaveCount(0)
    await expect(page.getByRole('link', { name: /create my vault/i })).toHaveCount(0)
  })

  test('signing in again is not offered to someone already signed in', async ({ page }) => {
    await register(page)

    for (const path of ['/sign-in', '/register']) {
      await page.goto(path)
      await page.waitForURL('**/collection')
      expect(new URL(page.url()).pathname).toBe('/collection')
    }
  })

  test('a signed-out visitor is offered sign in', async ({ page }) => {
    await gotoReady(page, '/')
    await expect(page.getByRole('link', { name: /^sign in$/i })).toBeVisible()
    await expect(page.getByRole('link', { name: /create my vault/i }).first()).toBeVisible()
  })

  // FR-011: the session is gone, not merely forgotten.
  test('signing out ends the session', async ({ page }) => {
    await register(page)
    await page.getByTestId('sign-out').click()
    await page.waitForURL(/\/$/)

    await expect(page.getByRole('link', { name: /^sign in$/i })).toBeVisible()
    expect(await page.context().cookies()).toEqual(
      expect.not.arrayContaining([expect.objectContaining({ name: 'better-auth.session_token' })]),
    )
  })
})
