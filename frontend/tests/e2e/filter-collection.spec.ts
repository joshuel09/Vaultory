import { expect, test } from '@playwright/test'
import { addCollectible, gotoReady, signIn } from './support'

test.describe('filtering the collection', () => {
  test.beforeEach(async ({ page }) => signIn(page))

  // User Story 3: narrow to a status, then return to everything (FR-037, FR-038).
  test('narrows to one status and back again', async ({ page }) => {
    const stamp = Date.now()
    await addCollectible(page, `Owned ${stamp}`, 'owned')
    await addCollectible(page, `Preordered ${stamp}`, 'preordered')

    await gotoReady(page, '/collection')
    await page.getByText('Preordered', { exact: true }).first().click()
    await expect(page).toHaveURL(/status=preordered/)
    await expect(page.getByText(`Preordered ${stamp}`)).toBeVisible()
    await expect(page.getByText(`Owned ${stamp}`)).toHaveCount(0)

    // One action back to everything.
    await page.getByText('All', { exact: true }).click()
    await expect(page).not.toHaveURL(/status=/)
    await expect(page.getByText(`Owned ${stamp}`)).toBeVisible()
  })

  // FR-040: a filter that matched nothing is not an empty vault, and must not say so.
  test('a filter matching nothing is distinct from an empty vault', async ({ page }) => {
    await addCollectible(page, `Only Owned ${Date.now()}`, 'owned')
    await gotoReady(page, '/collection?status=sold')

    await expect(page.getByTestId('no-results')).toBeVisible()
    await expect(page.getByText(/nothing marked sold/i)).toBeVisible()
    await expect(page.getByTestId('empty-collection')).toHaveCount(0)
    await expect(page.getByText(/your vault is empty/i)).toHaveCount(0)
  })

  // FR-039: the active filter is visible, and survives a reload because it lives in the URL.
  test('the active filter survives a reload', async ({ page }) => {
    await addCollectible(page, `Persisted ${Date.now()}`, 'wishlist')
    await gotoReady(page, '/collection?status=wishlist')
    await page.reload()
    await expect(page).toHaveURL(/status=wishlist/)
    await expect(page.getByText(/\(active filter\)/i)).toBeAttached()
  })
})
