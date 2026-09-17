import { expect, test } from '@playwright/test'
import { addCollectible, gotoReady, signIn } from './support'

/**
 * FR-036, SC-010: legible and usable at desktop, tablet, and mobile widths, with no horizontal
 * page scrolling and nothing clipped or overlapping.
 */
const WIDTHS = [
  { name: 'small phone', width: 360, height: 800 },
  { name: 'phone', width: 390, height: 844 },
  { name: 'tablet portrait', width: 820, height: 1180 },
  { name: 'tablet landscape', width: 1180, height: 820 },
  { name: 'laptop', width: 1440, height: 900 },
  { name: 'wide desktop', width: 1920, height: 1080 },
]

test.describe('responsive layout', () => {
  test.beforeEach(async ({ page }) => {
    await signIn(page)
    await addCollectible(page, `Responsive ${Date.now()}`)
  })

  for (const { name, width, height } of WIDTHS) {
    test(`the gallery holds at ${name} (${width}px)`, async ({ page }) => {
      await page.setViewportSize({ width, height })
      await gotoReady(page, '/collection')
      await expect(page.getByTestId('collection-gallery')).toBeVisible()

      const overflows = await page.evaluate(
        () => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1,
      )
      expect(overflows, 'the page must never scroll horizontally').toBe(false)

      // A card must not be clipped by the viewport.
      const card = page.getByTestId('collectible-card').first()
      const box = await card.boundingBox()
      expect(box).not.toBeNull()
      expect(box!.x).toBeGreaterThanOrEqual(0)
      expect(box!.x + box!.width).toBeLessThanOrEqual(width + 1)
    })

    test(`the add form holds at ${name} (${width}px)`, async ({ page }) => {
      await page.setViewportSize({ width, height })
      await gotoReady(page, '/collection/new')
      await expect(page.getByLabel(/^name/i)).toBeVisible()
      const overflows = await page.evaluate(
        () => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1,
      )
      expect(overflows, 'the form must never scroll horizontally').toBe(false)
    })
  }
})

/**
 * FR-045, SC-011: adding and browsing are operable entirely by keyboard.
 */
test.describe('keyboard operation', () => {
  test.beforeEach(async ({ page }) => signIn(page))

  test('the add form can be completed by keyboard alone', async ({ page }) => {
    await gotoReady(page, '/collection/new')
    // The name field takes focus on arrival, so a keyboard user starts where they need to be.
    await page.keyboard.type(`Keyboard ${Date.now()}`)
    await expect(page.getByLabel(/^name/i)).not.toHaveValue('')

    // Tab to the submit control and activate it without touching a pointer.
    for (let i = 0; i < 30; i++) {
      const isSubmit = await page.evaluate(
        () => document.activeElement?.textContent?.match(/add to my vault/i) !== null &&
              document.activeElement?.tagName === 'BUTTON',
      )
      if (isSubmit) break
      await page.keyboard.press('Tab')
    }
    await page.keyboard.press('Enter')
    await expect(page.getByTestId('add-success')).toBeVisible()
  })

  test('every focused control shows a visible focus ring', async ({ page }) => {
    await gotoReady(page, '/collection/new')
    await page.keyboard.press('Tab')
    const outlineVisible = await page.evaluate(() => {
      const el = document.activeElement
      if (!el) return false
      const style = getComputedStyle(el)
      // The ring is drawn with a box-shadow ring utility or an outline; either counts.
      return style.outlineStyle !== 'none' || style.boxShadow !== 'none'
    })
    expect(outlineVisible, 'an invisible focus ring is the same as none (FR-045)').toBe(true)
  })
})
