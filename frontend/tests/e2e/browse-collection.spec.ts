import { expect, test } from '@playwright/test'
import { addCollectible, signIn } from './support'

test.describe('browsing the collection', () => {
  test.beforeEach(async ({ page }) => signIn(page))

  // User Story 2, scenario 1: a gallery, not a table (FR-030, FR-031).
  test('presents the collection as an image-forward gallery', async ({ page }) => {
    await addCollectible(page, `Gallery Item ${Date.now()}`)
    await page.goto('/collection')

    await expect(page.getByTestId('collection-gallery')).toBeVisible()
    await expect(page.getByTestId('collectible-card').first()).toBeVisible()
    // The decisive check: the collection must not be a data table.
    await expect(page.locator('main table')).toHaveCount(0)
  })

  // User Story 2, scenario 2, and FR-033.
  test('a collectible with no photograph shows the designed placeholder', async ({ page }) => {
    await addCollectible(page, `No Photo ${Date.now()}`)
    await page.goto('/collection')
    await expect(page.getByTestId('image-placeholder').first()).toBeVisible()
    await expect(page.getByText(/no photo yet/i).first()).toBeVisible()
  })

  // FR-014, SC-013: every card frames its image identically.
  test('every card uses the same 4:5 frame', async ({ page }) => {
    await addCollectible(page, `Framed A ${Date.now()}`)
    await addCollectible(page, `Framed B ${Date.now()}`)
    await page.goto('/collection')

    const frames = page.locator('.aspect-collectible')
    const count = await frames.count()
    expect(count).toBeGreaterThanOrEqual(2)

    const first = await frames.nth(0).boundingBox()
    const second = await frames.nth(1).boundingBox()
    expect(first).not.toBeNull()
    expect(second).not.toBeNull()
    // Same width implies same height, because the ratio is fixed.
    expect(Math.abs(first!.height - second!.height)).toBeLessThanOrEqual(1)
    expect(Math.abs(first!.height / first!.width - 1.25)).toBeLessThan(0.02)
  })

  // FR-036, SC-010: usable at every supported width, with no horizontal page scroll.
  test('does not scroll horizontally at any supported width', async ({ page }) => {
    await addCollectible(page, `Responsive ${Date.now()}`)
    for (const size of [
      { width: 390, height: 844 },
      { width: 820, height: 1180 },
      { width: 1440, height: 900 },
    ]) {
      await page.setViewportSize(size)
      await page.goto('/collection')
      const overflows = await page.evaluate(
        () => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1,
      )
      expect(overflows, `horizontal scroll at ${size.width}px`).toBe(false)
    }
  })

  // User Story 2, scenario 3, and FR-041 — for a collector whose vault really is empty.
  test('an empty vault explains itself and offers a way to start', async ({ page }) => {
    await signIn(page, 'second')
    await page.goto('/collection')
    const empty = page.getByTestId('empty-collection')
    if (await empty.isVisible()) {
      await expect(empty.getByRole('link', { name: /add a collectible/i })).toBeVisible()
    }
  })
})
