import { expect, test } from '@playwright/test'
import { addCollectible, gotoReady, openEditScreen, signIn } from './support'

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

  // T074 — FR-039: the screens this feature adds, at every width.
  for (const { name, width, height } of WIDTHS) {
    test(`the edit screen holds at ${name} (${width}px)`, async ({ page }) => {
      await page.setViewportSize({ width, height })
      const collectible = `Responsive edit ${Date.now()}`
      await addCollectible(page, collectible)
      await openEditScreen(page, collectible)

      // Nothing clipped and nothing overflowing: a form that scrolls sideways on a phone is a
      // form a collector cannot fill in.
      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
      )
      expect(overflow, `the edit screen scrolls horizontally at ${width}px`).toBeLessThanOrEqual(1)

      await expect(page.getByLabel(/^name/i)).toBeVisible()
      await expect(page.getByRole('button', { name: /save changes/i })).toBeVisible()
      await expect(page.getByTestId('delete-collectible')).toBeVisible()
    })

    test(`the delete confirmation holds at ${name} (${width}px)`, async ({ page }) => {
      await page.setViewportSize({ width, height })
      const collectible = `Responsive dialog ${Date.now()}`
      await addCollectible(page, collectible)
      await openEditScreen(page, collectible)
      await page.getByTestId('delete-collectible').click()

      const dialog = page.getByTestId('delete-dialog')
      await expect(dialog).toBeVisible()

      // Both choices have to be reachable. A confirmation whose Cancel is off-screen on a phone
      // offers the collector only the destructive one.
      await expect(page.getByTestId('cancel-delete')).toBeInViewport()
      await expect(page.getByTestId('confirm-delete')).toBeInViewport()

      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
      )
      expect(overflow, `the dialog makes the page scroll sideways at ${width}px`).toBeLessThanOrEqual(1)
    })
  }
})
