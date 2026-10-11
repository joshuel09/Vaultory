import { expect, test } from '@playwright/test'
import { addCollectible, gotoReady, openEditScreen, signIn } from './support'

test.describe('deleting a collectible', () => {
  test.beforeEach(async ({ page }) => signIn(page))

  // User Story 2's independent test, and the defect feature 001 could not fix: a duplicate.
  test('removes one of two identical entries and leaves the other', async ({ page }) => {
    const name = `Kaiju Sentinel ${Date.now()}`
    await addCollectible(page, name)
    await addCollectible(page, name)

    await gotoReady(page, '/collection')
    await expect(page.getByTestId('collectible-card')).toHaveCount(2)

    await openEditScreen(page, name)
    await page.getByTestId('delete-collectible').click()

    const dialog = page.getByTestId('delete-dialog')
    await expect(dialog).toBeVisible()
    // FR-023: it names the collectible and says plainly that this cannot be undone.
    await expect(dialog).toContainText(name)
    await expect(dialog).toContainText(/cannot be undone/i)

    await page.getByTestId('confirm-delete').click()
    await page.waitForURL(/\/collection(\?|$)/)

    // Reload before judging it. One gone, one left.
    await gotoReady(page, '/collection')
    await expect(page.getByTestId('collectible-card')).toHaveCount(1)
    await expect(page.getByText(name)).toBeVisible()
  })

  // The platform behaviours the unit suite had to stub, verified in a real browser.
  test('Escape and Keep it both leave the collectible alone', async ({ page }) => {
    const name = `Spared ${Date.now()}`
    await addCollectible(page, name)
    await openEditScreen(page, name)

    await page.getByTestId('delete-collectible').click()
    await expect(page.getByTestId('delete-dialog')).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(page.getByTestId('delete-dialog')).toBeHidden()

    await page.getByTestId('delete-collectible').click()
    await page.getByTestId('cancel-delete').click()
    await expect(page.getByTestId('delete-dialog')).toBeHidden()

    await gotoReady(page, '/collection')
    await expect(page.getByText(name)).toBeVisible()
  })

  // FR-023: the destructive choice is not what a stray keypress lands on.
  test('Enter on an untouched dialog deletes nothing', async ({ page }) => {
    const name = `Untouched ${Date.now()}`
    await addCollectible(page, name)
    await openEditScreen(page, name)

    await page.getByTestId('delete-collectible').click()
    await expect(page.getByTestId('delete-dialog')).toBeVisible()
    // Cancel holds the focus, and the form's method="dialog" makes Enter close rather than submit
    // the deletion.
    await expect(page.getByTestId('cancel-delete')).toBeFocused()
    await page.keyboard.press('Enter')
    await expect(page.getByTestId('delete-dialog')).toBeHidden()

    await gotoReady(page, '/collection')
    await expect(page.getByText(name)).toBeVisible()
  })

  // FR-022a: a destructive action one stray click away while browsing is how a collection gets
  // damaged by accident.
  test('no gallery entry offers a delete control', async ({ page }) => {
    await addCollectible(page, `Browsing ${Date.now()}`)
    await gotoReady(page, '/collection')

    await expect(page.getByTestId('collectible-card')).toHaveCount(1)
    await expect(page.getByTestId('delete-collectible')).toHaveCount(0)
    await expect(page.getByRole('button', { name: /delete/i })).toHaveCount(0)
  })

  test('deleting the last collectible shows the designed empty state', async ({ page }) => {
    const name = `Only One ${Date.now()}`
    await addCollectible(page, name)
    await openEditScreen(page, name)

    await page.getByTestId('delete-collectible').click()
    await page.getByTestId('confirm-delete').click()
    await page.waitForURL(/\/collection(\?|$)/)

    await gotoReady(page, '/collection')
    // An empty vault, not an empty grid.
    await expect(page.getByTestId('empty-collection')).toBeVisible()
  })

  test('deleting while a status filter is active refreshes that view', async ({ page }) => {
    const sold = `Sold Piece ${Date.now()}`
    const owned = `Owned Piece ${Date.now()}`
    await addCollectible(page, sold, 'sold')
    await addCollectible(page, owned)

    await gotoReady(page, '/collection?status=sold')
    await expect(page.getByTestId('collectible-card')).toHaveCount(1)

    await page.getByTestId('collectible-card').first().getByTestId('edit-collectible').click()
    await page.waitForURL(/\/collection\/[^/]+\/edit/)
    await page.waitForLoadState('networkidle')
    await page.getByTestId('delete-collectible').click()
    await page.getByTestId('confirm-delete').click()

    // Back to the filter they were browsing, now matching nothing — the no-results state, not the
    // empty-vault one, because the collection is not empty.
    await page.waitForURL(/\/collection\?status=sold/)
    await expect(page.getByTestId('no-results')).toBeVisible()

    await gotoReady(page, '/collection')
    await expect(page.getByText(owned)).toBeVisible()
    await expect(page.getByText(sold)).toHaveCount(0)
  })
})
