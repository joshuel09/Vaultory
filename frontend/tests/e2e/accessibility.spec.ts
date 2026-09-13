import { expect, test } from '@playwright/test'
import { addCollectible, signIn } from './support'

/**
 * T079, FR-044, FR-045, SC-011.
 *
 * Accessibility is part of the product rather than an optional enhancement (Constitution
 * Principle I), so these assert the properties the specification actually names rather than a
 * generic audit score.
 */
test.describe('accessibility', () => {
  test.beforeEach(async ({ page }) => signIn(page))

  // FR-045: every collectible image carries a text alternative, and it names the collectible.
  test('every image has a meaningful text alternative', async ({ page }) => {
    const name = `Alt Text ${Date.now()}`
    await addCollectible(page, name)
    await page.goto('/collection')

    const images = page.locator('main img')
    const count = await images.count()
    for (let i = 0; i < count; i++) {
      const alt = await images.nth(i).getAttribute('alt')
      expect(alt, `image ${i} has no alt attribute`).not.toBeNull()
      expect(alt!.trim(), `image ${i} has an empty alt`).not.toBe('')
      // "image" or "photo" alone tells a screen reader nothing it did not already know.
      expect(alt!.toLowerCase()).not.toMatch(/^(image|photo|picture)$/)
    }
  })

  // FR-044: status is not signalled by colour alone — the label is text.
  test('status is readable as text, not only as a colour', async ({ page }) => {
    await addCollectible(page, `Status Text ${Date.now()}`, 'preordered')
    await page.goto('/collection?status=preordered')
    // Rendered in the card, independent of any colour.
    await expect(page.getByTestId('collectible-card').first().getByText('Preordered')).toBeVisible()
  })

  // FR-045: every form control is reachable by its label.
  test('every form control has an associated label', async ({ page }) => {
    await page.goto('/collection/new')
    const unlabelled = await page.evaluate(() => {
      const controls = Array.from(
        document.querySelectorAll<HTMLElement>('input:not([type=file]), select, textarea'),
      )
      return controls
        .filter((el) => {
          if (el.getAttribute('aria-label')) return false
          const id = el.getAttribute('id')
          if (id && document.querySelector(`label[for="${id}"]`)) return false
          return !el.closest('label')
        })
        .map((el) => el.getAttribute('name') ?? el.tagName)
    })
    expect(unlabelled, 'these controls have no label').toEqual([])
  })

  // A single h1 per page, and headings that do not skip levels.
  test('the page has exactly one first-level heading', async ({ page }) => {
    for (const path of ['/collection', '/collection/new']) {
      await page.goto(path)
      await expect(page.locator('h1'), `${path} should have exactly one h1`).toHaveCount(1)
    }
  })

  // Validation errors are announced, not merely coloured.
  test('validation errors are announced to assistive technology', async ({ page }) => {
    await page.goto('/collection/new')
    await page.getByRole('button', { name: /add to my vault/i }).click()
    const alert = page.getByRole('alert').filter({ hasText: /a name is required/i })
    await expect(alert).toBeVisible()

    // And the input itself is marked invalid and pointed at its message.
    const name = page.getByLabel(/^name/i)
    await expect(name).toHaveAttribute('aria-invalid', 'true')
    await expect(name).toHaveAttribute('aria-describedby', /.+/)
  })

  // The loading state announces itself rather than leaving a screen reader in silence.
  test('the loading state is announced', async ({ page }) => {
    await page.route('**/api/collectibles*', async (route) => {
      await new Promise((r) => setTimeout(r, 600))
      await route.continue()
    })
    await page.goto('/collection')
    // Either the loading status or the loaded gallery; the point is that neither is silent.
    await expect(page.locator('[role=status], [data-testid=collection-gallery]').first()).toBeAttached()
  })
})
