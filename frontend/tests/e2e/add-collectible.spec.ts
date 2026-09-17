import { expect, test } from '@playwright/test'
import { addCollectible, gotoReady, notAnImage, oversizedFile, signIn } from './support'

test.describe('adding a collectible', () => {
  test.beforeEach(async ({ page }) => signIn(page))

  // User Story 1, scenario 1. The MVP proof: a name and a status are enough.
  test('a name and a status alone are enough', async ({ page }) => {
    const name = `Kaiju Sentinel ${Date.now()}`
    await addCollectible(page, name)

    await gotoReady(page, '/collection')
    await expect(page.getByText(name)).toBeVisible()
  })

  // User Story 1, scenario 2: every attribute recorded, including a photograph.
  test('records every attribute', async ({ page }) => {
    const name = `Full Detail ${Date.now()}`
    await gotoReady(page, '/collection/new')
    await page.getByLabel(/^name/i).fill(name)
    await page.getByLabel(/character/i).fill('Sentinel Prime')
    await page.getByLabel(/series or franchise/i).fill('Kaiju Wars')
    await page.getByLabel(/manufacturer/i).fill('Apex Studio')
    await page.getByLabel(/category/i).fill('Statue')
    await page.getByLabel(/scale/i).fill('1/4')
    await page.getByLabel(/edition or variant/i).fill('Deluxe Exclusive')
    await page.getByLabel(/purchase price/i).fill('1250.00')
    await page.getByLabel(/purchase date/i).fill('2026-08-14')
    await page.getByLabel(/release date/i).fill('2026-11-30')
    await page.getByLabel(/notes/i).fill('Box has a small dent on the lower left corner.')

    await page.getByRole('button', { name: /add to my vault/i }).click()
    await expect(page.getByTestId('add-success')).toBeVisible()

    await gotoReady(page, '/collection')
    await expect(page.getByText(name)).toBeVisible()
    await expect(page.getByText('Kaiju Wars').first()).toBeVisible()
  })

  // User Story 1, scenarios 3 and 4, and FR-020: everything wrong, reported together.
  test('reports every validation problem at once', async ({ page }) => {
    await gotoReady(page, '/collection/new')
    await page.getByLabel(/^name/i).fill('   ')
    await page.getByLabel(/purchase price/i).fill('-5.00')
    await page.getByLabel(/purchase date/i).fill('2099-01-01')
    await page.getByRole('button', { name: /add to my vault/i }).click()

    await expect(page.getByText(/a name is required/i)).toBeVisible()
    await expect(page.getByText(/cannot be negative/i)).toBeVisible()
    await expect(page.getByText(/cannot be in the future/i)).toBeVisible()
  })

  // User Story 1, scenario 6, and FR-013: a refused photo does not block the save.
  test('a 10 MB refusal does not stop the collectible being saved', async ({ page }) => {
    const name = `Refused Photo ${Date.now()}`
    await gotoReady(page, '/collection/new')
    await page.getByLabel(/^name/i).fill(name)

    await page.locator('#collectible-image').setInputFiles(oversizedFile)
    await expect(page.getByTestId('image-problem')).toContainText(/10 MB/i)

    // The name survived the refusal, and the collectible still saves without a photo.
    await expect(page.getByLabel(/^name/i)).toHaveValue(name)
    await page.getByRole('button', { name: /add to my vault/i }).click()
    await expect(page.getByTestId('add-success')).toBeVisible()
  })

  // FR-009: the filename and declared type claim JPEG; only decoding reveals otherwise.
  test('a non-image is refused by its content, not its name', async ({ page }) => {
    await gotoReady(page, '/collection/new')
    await page.getByLabel(/^name/i).fill('Wrong Format')
    await page.locator('#collectible-image').setInputFiles(notAnImage)
    await expect(page.getByTestId('image-problem')).toContainText(/JPEG, PNG, and WebP/i)
  })

  // FR-047 through the browser: a double-click must not create two collectibles.
  test('double-clicking save creates only one collectible', async ({ page }) => {
    const name = `Double Click ${Date.now()}`
    await gotoReady(page, '/collection/new')
    await page.getByLabel(/^name/i).fill(name)

    const save = page.getByRole('button', { name: /add to my vault/i })
    await save.click({ clickCount: 2, delay: 10 })
    await expect(page.getByTestId('add-success')).toBeVisible()

    await gotoReady(page, '/collection')
    await expect(page.getByText(name)).toHaveCount(1)
  })

  // FR-023: a deliberate second copy is a new submission and must appear twice.
  test('adding the same collectible twice deliberately keeps both', async ({ page }) => {
    const name = `Deliberate Duplicate ${Date.now()}`
    await addCollectible(page, name)
    await addCollectible(page, name)

    await gotoReady(page, '/collection')
    await expect(page.getByText(name)).toHaveCount(2)
  })
})
