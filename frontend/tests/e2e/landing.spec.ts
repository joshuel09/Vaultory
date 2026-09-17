import { expect, test } from '@playwright/test'
import { gotoReady } from './support'

/**
 * Feature 003. Every test here runs with no session on purpose: the point of this page is that a
 * visitor who has never signed in can read it (FR-001).
 */
test.describe('the landing page', () => {
  test.beforeEach(async ({ page }) => page.context().clearCookies())

  // FR-001, SC-001
  test('is public, and does not bounce a visitor into a private vault', async ({ page }) => {
    await gotoReady(page, '/')
    expect(new URL(page.url()).pathname).toBe('/')
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
    await expect(page.getByTestId('collection-error')).toHaveCount(0)
  })

  // FR-002, FR-005
  test('says what Vaultory is and shows the gallery', async ({ page }) => {
    await gotoReady(page, '/')
    await expect(page.getByRole('heading', { level: 1 })).toContainText(/collection/i)
    await expect(page.getByRole('list', { name: /example of a collection gallery/i })).toBeVisible()
  })

  // FR-003: the vocabulary the product is built on.
  test('names all four collection statuses', async ({ page }) => {
    await gotoReady(page, '/')
    const main = page.getByRole('main')
    for (const label of ['Owned', 'Preordered', 'Wishlist', 'Sold']) {
      await expect(main.getByText(label, { exact: true }).first()).toBeVisible()
    }
  })

  // FR-006, SC-002
  test('reaches a vault in one click', async ({ page }) => {
    await gotoReady(page, '/')
    await page.getByRole('link', { name: /open my vault/i }).first().click()
    await page.waitForURL('**/collection')
    expect(new URL(page.url()).pathname).toBe('/collection')
  })

  // FR-010, SC-005: it renders, it does not fetch.
  test('makes no request to the API', async ({ page }) => {
    const apiCalls: string[] = []
    page.on('request', (r) => { if (r.url().includes('/api/')) apiCalls.push(r.url()) })
    await gotoReady(page, '/')
    expect(apiCalls).toEqual([])
  })

  // FR-009: one h1, and headings that do not skip a level.
  test('has a single h1 and ordered headings', async ({ page }) => {
    await gotoReady(page, '/')
    await expect(page.getByRole('heading', { level: 1 })).toHaveCount(1)
    const levels = await page.evaluate(() =>
      [...document.querySelectorAll('h1,h2,h3,h4')].map((h) => Number(h.tagName[1])),
    )
    for (let i = 1; i < levels.length; i++) {
      expect(levels[i]! - levels[i - 1]!).toBeLessThanOrEqual(1)
    }
  })

  // FR-007, SC-003
  for (const { name, width, height } of [
    { name: 'small phone', width: 360, height: 800 },
    { name: 'phone', width: 390, height: 844 },
    { name: 'tablet portrait', width: 820, height: 1180 },
    { name: 'tablet landscape', width: 1180, height: 820 },
    { name: 'laptop', width: 1440, height: 900 },
    { name: 'wide desktop', width: 1920, height: 1080 },
  ]) {
    test(`does not scroll horizontally at ${name} (${width}px)`, async ({ page }) => {
      await page.setViewportSize({ width, height })
      await gotoReady(page, '/')
      const overflows = await page.evaluate(
        () => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1,
      )
      expect(overflows, 'the landing page must never scroll horizontally').toBe(false)
    })
  }
})

// FR-008, SC-004: both appearances, from the same tokens.
for (const scheme of ['dark', 'light'] as const) {
  test(`renders in ${scheme} appearance`, async ({ browser }) => {
    const context = await browser.newContext({ colorScheme: scheme })
    const page = await context.newPage()
    await page.goto('/')
    await page.waitForLoadState('networkidle')
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
    const ground = await page.evaluate(() => getComputedStyle(document.body).backgroundColor)
    expect(ground).toMatch(/^rgba?\(/)
    await context.close()
  })
}
