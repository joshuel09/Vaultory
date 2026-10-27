import { expect, test } from '@playwright/test'
import { addCollectible, gotoReady, signIn } from './support'

// Feature 011: find one piece without scrolling, and order the gallery.

test.describe('search and sort', () => {
  test.beforeEach(async ({ page }) => signIn(page))

  test('a search finds one collectible and survives editing it (US1, US3)', async ({ page }) => {
    const stamp = Date.now()
    const wanted = `Zeppelin Sentinel ${stamp}`
    await addCollectible(page, wanted)
    await addCollectible(page, `Kaiju Other ${stamp}`)

    await gotoReady(page, '/collection')
    await page.getByRole('searchbox', { name: 'Search your vault' }).fill(`zeppelin ${stamp}`)
    await page.keyboard.press('Enter')
    await page.waitForURL(/[?&]q=/)

    await expect(page.getByTestId('collectible-card')).toHaveCount(1)
    await expect(page.getByText(wanted)).toBeVisible()

    // Edit and save: the gallery comes back searched.
    await openEditScreenFromCurrent(page, wanted)
    await page.getByRole('button', { name: /save changes/i }).click()
    await page.waitForURL(/\/collection\?.*q=/)
    await expect(page.getByTestId('collectible-card')).toHaveCount(1)
  })

  test('a search that matches nothing says so, and is not the empty vault (FR-005)', async ({ page }) => {
    await addCollectible(page, `Present ${Date.now()}`)
    await gotoReady(page, `/collection?q=nothing-${Date.now()}`)
    await expect(page.getByTestId('no-results')).toContainText('Nothing matching')
    await expect(page.getByTestId('empty-collection')).toHaveCount(0)
  })

  test('the order menu reorders the gallery and stays in the address (US2)', async ({ page }) => {
    const stamp = Date.now()
    await addCollectible(page, `Aardvark ${stamp}`)
    await addCollectible(page, `Zebra ${stamp}`)

    await gotoReady(page, `/collection?q=${stamp}`)
    await page.getByRole('combobox', { name: 'Sort by' }).selectOption('name_asc')
    await page.waitForURL(/sort=name_asc/)
    await expect(page.getByTestId('collectible-card').first()).toContainText('Aardvark')

    await page.getByRole('combobox', { name: 'Sort by' }).selectOption('name_desc')
    await page.waitForURL(/sort=name_desc/)
    await expect(page.getByTestId('collectible-card').first()).toContainText('Zebra')
  })
})

// openEditScreen in support navigates to the unfiltered gallery first; this keeps the search.
async function openEditScreenFromCurrent(page: import('@playwright/test').Page, name: string) {
  const card = page.getByTestId('collectible-card').filter({ hasText: name }).first()
  await card.getByTestId('edit-collectible').click()
  await page.waitForURL(/\/collection\/[^/]+\/edit/)
  await page.waitForLoadState('networkidle')
}

