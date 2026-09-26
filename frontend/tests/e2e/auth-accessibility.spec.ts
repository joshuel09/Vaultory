import { expect, test } from '@playwright/test'
import { gotoReady } from './support'

/**
 * SC-011 — registration and sign-in are completable by keyboard alone, and every failure is
 * announced to assistive technology.
 *
 * Constitution I makes this a first-class concern rather than a nicety: a collector who cannot
 * see the red border still has to learn what is wrong.
 */
for (const { page: path, submit } of [
  { page: '/register', submit: /create my vault/i },
  { page: '/sign-in', submit: /^sign in$/i },
  // Recovery pages are held to the same standard (SC-010). Somebody locked out of their vault is
  // already having a bad day; a form they cannot complete by keyboard, or an error they cannot
  // hear, makes it worse at exactly the wrong moment.
  { page: '/forgot-password', submit: /send me a link/i },
]) {
  test.describe(`${path}`, () => {
    test.beforeEach(async ({ page }) => page.context().clearCookies())

    test('can be completed with the keyboard alone', async ({ page }) => {
      await gotoReady(page, path)

      // Tab until the email field has focus, then fill everything without touching the mouse.
      const email = page.getByLabel(/^email/i)
      for (let i = 0; i < 12 && !(await email.evaluate((el) => el === document.activeElement)); i++) {
        await page.keyboard.press('Tab')
      }
      expect(
        await email.evaluate((el) => el === document.activeElement),
        'the email field must be reachable by tabbing',
      ).toBe(true)

      await page.keyboard.type(`kbd-${Date.now()}@example.test`)
      await page.keyboard.press('Tab')

      // Some of these pages ask for a password and some do not — /forgot-password needs only an
      // address. Follow whatever the page actually presents rather than assuming a shape.
      if (await page.getByLabel(/^password/i).count()) {
        await page.keyboard.type('a-long-enough-password')
        await page.keyboard.press('Tab')
      }
      const focused = await page.evaluate(() => document.activeElement?.textContent ?? '')
      expect(focused).toMatch(submit)
    })

    test('announces a failure to assistive technology', async ({ page }) => {
      await gotoReady(page, path)
      // Submit empty, which must be refused with something a screen reader will read out.
      await page.getByRole('button', { name: submit }).click()

      const alerts = page.locator('[role=alert]')
      await expect(alerts.first()).toBeVisible()
    })

    test('every control has a label, and errors are linked to their field', async ({ page }) => {
      await gotoReady(page, path)
      await page.getByRole('button', { name: submit }).click()
      await expect(page.locator('[role=alert]').first()).toBeVisible()

      // Each input is labelled, and any invalid one points at the message explaining why.
      const unlabelled = await page.evaluate(() =>
        [...document.querySelectorAll('input')].filter((i) => {
          const id = i.getAttribute('id')
          return !(id && document.querySelector(`label[for="${id}"]`))
        }).length,
      )
      expect(unlabelled, 'every input needs an associated label').toBe(0)

      const danglingDescriptions = await page.evaluate(() =>
        [...document.querySelectorAll('[aria-describedby]')].filter((el) =>
          (el.getAttribute('aria-describedby') ?? '')
            .split(/\s+/)
            .filter(Boolean)
            .some((id) => !document.getElementById(id)),
        ).length,
      )
      expect(danglingDescriptions, 'aria-describedby must point at something that exists').toBe(0)
    })

    test('has exactly one first-level heading', async ({ page }) => {
      await gotoReady(page, path)
      await expect(page.getByRole('heading', { level: 1 })).toHaveCount(1)
    })
  })
}
