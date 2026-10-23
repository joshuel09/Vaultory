import { expect, test } from '@playwright/test'
import { addCollectible, fillWhenReady, gotoReady, largestPermittedJpeg, notAnImage, oversizedFile, signIn } from './support'

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
    await fillWhenReady(page.getByLabel(/^name/i), name)
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
    await fillWhenReady(page.getByLabel(/^name/i), '   ')
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
    await fillWhenReady(page.getByLabel(/^name/i), name)

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
    await fillWhenReady(page.getByLabel(/^name/i), 'Wrong Format')
    await page.locator('#collectible-image').setInputFiles(notAnImage)
    await expect(page.getByTestId('image-problem')).toContainText(/JPEG, PNG, and WebP/i)
  })

  // FR-047 through the browser: a double-click must not create two collectibles.
  test('double-clicking save creates only one collectible', async ({ page }) => {
    const name = `Double Click ${Date.now()}`
    await gotoReady(page, '/collection/new')
    await fillWhenReady(page.getByLabel(/^name/i), name)

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

/*
 * Issue #47 — the upload path through the frontend origin, which nothing else exercised.
 *
 * Every other test in this suite either drives ImagePicker, whose client-side size check fires
 * before any request is made, or talks to Go directly, where the proxy is not in the picture. The
 * defect lived in the gap between the two: Next truncated a proxied request body at 10 MiB and
 * left the stream open, so Go waited for the rest of a Content-Length that never arrived until
 * proxyTimeout fired sixty seconds later and the client got a 500.
 *
 * These go through page.request, so they cross the rewrite exactly as a non-browser client would,
 * and they assert the thing that was wrong: the status, and that it arrives promptly. The sixty
 * seconds is itself the regression signal — if this breaks again the request outlives Playwright's
 * timeout and the test fails on time rather than on status.
 */
test.describe('uploading through the frontend origin', () => {
  test.beforeEach(async ({ page }) => signIn(page))

  test('refuses an oversized image with the documented 413, and does not hang', async ({ page }) => {
    const started = Date.now()
    const response = await page.request.post('/api/images', {
      multipart: { file: oversizedFile },
      timeout: 25_000,
    })
    const elapsed = Date.now() - started

    expect(response.status(), 'an oversized upload must answer 413, not 500').toBe(413)
    const body = await response.json()
    expect(body.error.code).toBe('image_too_large')
    expect(body.error.message).toMatch(/10 MB/i)

    // Comfortably inside proxyTimeout. A truncated body would take sixty seconds.
    expect(elapsed, 'the refusal must be prompt, not a proxy timeout').toBeLessThan(20_000)
  })

  test('accepts an image at the top of the permitted range', async ({ page }) => {
    // The half of #47 that was silently broken: 10 MiB is FR-010's limit *and* was the proxy's
    // cap, so the largest legitimate upload failed exactly like an oversized one.
    const response = await page.request.post('/api/images', {
      multipart: { file: largestPermittedJpeg() },
      timeout: 25_000,
    })

    expect(response.status(), 'the largest permitted image must upload, not time out').toBe(201)
    const body = await response.json()
    expect(body.id).toBeTruthy()
    expect(body.renditionUrl).toContain(body.id)
  })
})
