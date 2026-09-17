import { expect, test, type Page } from '@playwright/test'
import { addCollectible, gotoReady, signIn } from './support'

/**
 * FR-046, SC-015: every screen and every interface state is legible in both appearances.
 *
 * Dark is the default and light follows the collector's system preference, so these run the same
 * journey twice under an emulated preference. A state that only exists in one appearance is the
 * failure this is looking for.
 */

/** The computed background of the page body, as a crude but reliable appearance probe. */
async function bodyBackground(page: Page): Promise<string> {
  return page.evaluate(() => getComputedStyle(document.body).backgroundColor)
}

for (const scheme of ['dark', 'light'] as const) {
  test.describe(`${scheme} appearance`, () => {
    test.use({ colorScheme: scheme })

    test.beforeEach(async ({ page }) => signIn(page))

    test('the gallery renders with the expected ground', async ({ page }) => {
      await addCollectible(page, `Appearance ${scheme} ${Date.now()}`)
      await gotoReady(page, '/collection')
      await expect(page.getByTestId('collection-gallery')).toBeVisible()

      const background = await bodyBackground(page)
      expect(background, 'the body must paint its own ground, not inherit one').not.toBe('rgba(0, 0, 0, 0)')
    })

    test('the add form and its validation state are legible', async ({ page }) => {
      await gotoReady(page, '/collection/new')
      await expect(page.getByLabel(/^name/i)).toBeVisible()

      // The validation state, in this appearance.
      await page.getByRole('button', { name: /add to my vault/i }).click()
      const error = page.getByText(/a name is required/i)
      await expect(error).toBeVisible()
      const colour = await error.evaluate((el) => getComputedStyle(el).color)
      expect(colour).not.toBe('rgba(0, 0, 0, 0)')
    })

    test('the success state is legible', async ({ page }) => {
      await addCollectible(page, `Success ${scheme} ${Date.now()}`)
      await expect(page.getByTestId('add-success')).toBeVisible()
    })

    test('the no-results state is legible', async ({ page }) => {
      await addCollectible(page, `Filtered ${scheme} ${Date.now()}`, 'owned')
      await gotoReady(page, '/collection?status=sold')
      await expect(page.getByTestId('no-results')).toBeVisible()
    })

    test('the error state is legible', async ({ page }) => {
      // Force the collection request to fail so the error boundary renders.
      //
      // By dropping the session, not by intercepting the request. The first page of a collection
      // is fetched by a Server Component, so it never travels through the browser and page.route
      // cannot see it — aborting '**/api/collectibles*' here changed nothing and the page rendered
      // normally. Without a session that server-side fetch gets a 401, which is what the route
      // throws on, and the error boundary renders for real.
      await page.context().clearCookies()
      await gotoReady(page, '/collection')
      await expect(page.getByTestId('collection-error')).toBeVisible()
      await expect(page.getByRole('button', { name: /try again/i })).toBeVisible()
    })
  })
}

/**
 * The two appearances must actually differ. A single palette applied to both would pass every
 * assertion above while failing the requirement entirely.
 */
test('dark and light are genuinely different', async ({ browser }) => {
  const dark = await browser.newContext({ colorScheme: 'dark' })
  const light = await browser.newContext({ colorScheme: 'light' })
  const darkPage = await dark.newPage()
  const lightPage = await light.newPage()

  await signIn(darkPage)
  await signIn(lightPage)
  await darkPage.goto('/collection')
  await lightPage.goto('/collection')

  expect(await bodyBackground(darkPage)).not.toBe(await bodyBackground(lightPage))

  await dark.close()
  await light.close()
})
