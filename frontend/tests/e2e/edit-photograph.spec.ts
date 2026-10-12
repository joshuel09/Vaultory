import { expect, test } from '@playwright/test'
import {
  firstPhoto,
  gotoReady,
  openEditScreen,
  oversizedFile,
  replacementPhoto,
  signIn,
} from './support'

test.describe("changing a collectible's photograph", () => {
  test.beforeEach(async ({ page }) => signIn(page))

  /**
   * The rendition URL on a collectible's card, once the gallery has rendered it.
   *
   * Deliberately not gotoReady: its networkidle wait is for hydration before typing, and these
   * checks only read. On a gallery of lazily-loaded images behind a dev-mode websocket, networkidle
   * is a signal that may never arrive — waiting for the element itself is both more reliable and
   * closer to what the assertion is about.
   */
  async function renditionUrlOf(page: import('@playwright/test').Page, name: string) {
    await page.goto('/collection')
    const img = page.getByTestId('collectible-card').filter({ hasText: name }).first().locator('img')
    await expect(img).toBeVisible()
    return (await img.getAttribute('src')) ?? ''
  }

  /** Add a collectible carrying a real photograph, and return its rendition URL. */
  async function addWithPhoto(page: import('@playwright/test').Page, name: string) {
    await gotoReady(page, '/collection/new')
    await page.getByLabel(/^name/i).fill(name)
    await page.locator('#collectible-image').setInputFiles(firstPhoto)
    await expect(page.getByRole('button', { name: /remove/i })).toBeVisible()
    await page.getByRole('button', { name: /add to my vault/i }).click()
    await expect(page.getByTestId('add-success')).toBeVisible()
    await page.waitForLoadState('networkidle')

    return renditionUrlOf(page, name)
  }

  // User Story 3's independent test.
  test('replaces a photograph, and the old one stops being readable', async ({ page }) => {
    const name = `Badly Shot ${Date.now()}`
    const originalUrl = await addWithPhoto(page, name)
    expect(originalUrl).not.toBe('')

    // Readable by its owner beforehand.
    expect((await page.request.get(originalUrl)).status()).toBe(200)

    await openEditScreen(page, name)
    await page.locator('#collectible-image').setInputFiles(replacementPhoto)
    // Wait for the upload to finish before saving. Uploading is a separate operation from saving,
    // so without this the form is still holding the old photograph when the save goes out — which
    // is exactly the race the Save button is now disabled for.
    await expect(page.getByTestId('image-preview')).toHaveAttribute('data-uploading', 'false')
    await expect(page.locator('#collectible-image')).toBeEnabled()
    await page.getByRole('button', { name: /save changes/i }).click()
    await page.waitForURL(/\/collection(\?|$)/)

    // The card now shows a different rendition. Polled rather than read once, because the save
    // navigates and the gallery re-renders on arrival.
    const newUrl = await expect
      .poll(() => renditionUrlOf(page, name), { timeout: 15_000 })
      .not.toBe(originalUrl)
      .then(() => renditionUrlOf(page, name))

    /*
     * The old rendition is gone, by its owner's own session.
     *
     * Deleting the image row is what achieves this. Unlinking would not: a rendition is authorized
     * against the image's own collector_id, because the upload preview has to work before any
     * collectible references it (FR-020).
     */
    expect((await page.request.get(originalUrl)).status()).toBe(404)
    expect((await page.request.get(newUrl ?? '')).status()).toBe(200)
  })

  test('removes a photograph, and the placeholder takes its place', async ({ page }) => {
    const name = `Unphotographed ${Date.now()}`
    const url = await addWithPhoto(page, name)

    await openEditScreen(page, name)
    await page.getByRole('button', { name: /remove/i }).click()
    await page.getByRole('button', { name: /save changes/i }).click()
    await page.waitForURL(/\/collection(\?|$)/)

    await page.goto('/collection')
    const card = page.getByTestId('collectible-card').filter({ hasText: name }).first()
    // The designed placeholder, not a broken image (FR-017).
    await expect(card.locator('img')).toHaveCount(0)
    await expect(card.getByTestId('image-placeholder')).toBeVisible()

    expect((await page.request.get(url)).status()).toBe(404)
  })

  test('a refused replacement leaves the original in place', async ({ page }) => {
    const name = `Kept ${Date.now()}`
    const url = await addWithPhoto(page, name)

    await openEditScreen(page, name)
    await page.locator('#collectible-image').setInputFiles(oversizedFile)
    await expect(page.getByTestId('image-problem')).toContainText(/10 MB/i)

    // The collector can still save, and the photograph they already had is untouched (FR-019).
    await page.getByLabel(/^name/i).fill(`${name} renamed`)
    await page.getByRole('button', { name: /save changes/i }).click()
    await page.waitForURL(/\/collection(\?|$)/)

    expect((await page.request.get(url)).status()).toBe(200)
    await page.goto('/collection')
    await expect(
      page.getByTestId('collectible-card').filter({ hasText: `${name} renamed` }).first().locator('img'),
    ).toBeVisible()
  })

  // The sharpest edge in the contract, from the collector's side: PUT is a full replacement, so a
  // client that omits imageId destroys the photograph. An edit that touches nothing else must not.
  test('an edit that only changes the name keeps the photograph', async ({ page }) => {
    const name = `Typo ${Date.now()}`
    const url = await addWithPhoto(page, name)

    await openEditScreen(page, name)
    await page.getByLabel(/^name/i).fill(`${name} corrected`)
    await page.getByRole('button', { name: /save changes/i }).click()
    await page.waitForURL(/\/collection(\?|$)/)

    expect((await page.request.get(url)).status()).toBe(200)
    await page.goto('/collection')
    await expect(
      page.getByTestId('collectible-card').filter({ hasText: `${name} corrected` }).first().locator('img'),
    ).toBeVisible()
  })
})
