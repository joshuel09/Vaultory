import { expect, test } from '@playwright/test'
import { addCollectible, addDetailedCollectible, fillWhenReady, gotoReady, openEditScreen, seedCollectible, signIn } from './support'

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
    await gotoReady(page, '/collection')

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
    await gotoReady(page, '/collection?status=preordered')
    // Rendered in the card, independent of any colour.
    await expect(page.getByTestId('collectible-card').first().getByText('Preordered')).toBeVisible()
  })

  // FR-045: every form control is reachable by its label.
  test('every form control has an associated label', async ({ page }) => {
    await gotoReady(page, '/collection/new')
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
    await gotoReady(page, '/collection/new')
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
    await gotoReady(page, '/collection')
    // The loading status, the loaded gallery, or the empty state — the point is that the page
    // never leaves a screen reader with nothing. The empty state belongs in that list now:
    // feature 004 gives every account its own vault, so a fresh one genuinely has no collectibles,
    // where this test used to inherit whatever the shared seeded collector happened to hold.
    await expect(
      page
        .locator('[role=status], [data-testid=collection-gallery], [data-testid=empty-collection]')
        .first(),
    ).toBeAttached()
  })

  // T072 — FR-037: editing and deleting are fully operable by keyboard, and the dialog behaves
  // like a dialog.
  test('the edit screen is reachable from the gallery by keyboard', async ({ page }) => {
    const name = `Keyboard ${Date.now()}`
    await addCollectible(page, name)
    await gotoReady(page, '/collection')

    const edit = page.getByTestId('edit-collectible').first()
    await edit.focus()
    await expect(edit).toBeFocused()
    await page.keyboard.press('Enter')

    await page.waitForURL(/\/collection\/[^/]+\/edit/)
    await expect(page.getByLabel(/^name/i)).toBeVisible()
  })

  test('the delete confirmation traps focus and restores it on close', async ({ page }) => {
    const name = `Dialog ${Date.now()}`
    await addCollectible(page, name)
    await openEditScreen(page, name)

    const opener = page.getByTestId('delete-collectible')
    await opener.focus()
    await page.keyboard.press('Enter')
    await expect(page.getByTestId('delete-dialog')).toBeVisible()

    // Focus moves into the dialog, onto the way out rather than the way through.
    await expect(page.getByTestId('cancel-delete')).toBeFocused()

    /*
     * What showModal() actually guarantees, and what matters here: everything behind the dialog is
     * inert. Not that Tab cycles within the dialog's own children — Tab legitimately passes
     * through the browser's own chrome, so document.activeElement becomes <body> for a step, and
     * asserting otherwise tests the browser rather than the product.
     *
     * The failure this is looking for is a collector tabbing out of a confirmation and into the
     * form behind it, editing a collectible that is halfway through being deleted.
     */
    for (let i = 0; i < 10; i++) {
      await page.keyboard.press('Tab')
      const landedBehind = await page.evaluate(() => {
        const active = document.activeElement
        if (!active || active === document.body || active === document.documentElement) return null
        const dialog = document.querySelector('[data-testid=delete-dialog]')
        if (dialog?.contains(active)) return null
        return active.tagName + (active.getAttribute('name') ?? active.id ?? '')
      })
      expect(landedBehind, `Tab ${i + 1} reached ${landedBehind}, which is behind the dialog`)
        .toBeNull()
    }

    // The same property, asserted directly rather than by tabbing: the form behind cannot take
    // focus even when asked.
    const focusedBehind = await page.evaluate(() => {
      const field = document.querySelector<HTMLInputElement>('input[id$=name], input')
      field?.focus()
      const dialog = document.querySelector('[data-testid=delete-dialog]')
      return field !== null && document.activeElement === field && !dialog?.contains(field)
    })
    expect(focusedBehind, 'a field behind the dialog took focus; the dialog is not modal').toBe(false)

    await page.keyboard.press('Escape')
    await expect(page.getByTestId('delete-dialog')).toBeHidden()
    // And back where it came from, so a keyboard user is not dropped at the top of the document.
    await expect(opener).toBeFocused()
  })

  test('the dialog names the collectible for assistive technology', async ({ page }) => {
    const name = `Named ${Date.now()}`
    await addCollectible(page, name)
    await openEditScreen(page, name)
    await page.getByTestId('delete-collectible').click()

    const dialog = page.getByTestId('delete-dialog')
    await expect(dialog).toHaveAttribute('aria-labelledby', /.+/)
    const labelId = await dialog.getAttribute('aria-labelledby')
    await expect(page.locator(`#${labelId}`)).toContainText(name)
  })

  test('a version conflict is announced rather than shown silently', async ({ page, context }) => {
    const name = `Announced ${Date.now()}`
    await addCollectible(page, name)
    await openEditScreen(page, name)
    const staleUrl = page.url()

    const other = await context.newPage()
    await other.goto(staleUrl)
    await other.waitForLoadState('networkidle')
    await fillWhenReady(other.getByLabel(/^name/i), `${name} elsewhere`)
    await other.getByRole('button', { name: /save changes/i }).click()
    await other.waitForURL(/\/collection(\?|$)/)
    await other.close()

    await page.getByRole('button', { name: /save changes/i }).click()
    // role=alert, so a screen reader is told the save did not land rather than being left to
    // wonder why nothing happened.
    await expect(page.getByTestId('version-conflict')).toHaveAttribute('role', 'alert')
  })
})

/**
 * T070 — FR-036, SC-009: the statistics view by keyboard alone, at every viewport.
 *
 * It runs under all three Playwright projects rather than desktop only, because the screen reflows
 * between them and a control that is reachable in a four-column grid is not automatically reachable
 * in a one-column one.
 */
test.describe('the statistics view by keyboard', () => {
  test.beforeEach(async ({ page }) => signIn(page))

  test('every figure is reachable by keyboard and says what it leads to', async ({ page }) => {
    await addDetailedCollectible(page, {
      name: `A11y stats ${Date.now()}`,
      status: 'owned',
      price: '60.00',
      series: 'Kaiju Wars',
      manufacturer: 'Apex Studio',
    })
    await gotoReady(page, '/collection/stats')

    // Every link has an accessible name. "3" alone tells a screen-reader user nothing about what
    // following it would do.
    const links = page.locator('main a, body a')
    const count = await links.count()
    expect(count).toBeGreaterThan(0)
    for (let i = 0; i < count; i++) {
      const name = (await links.nth(i).textContent())?.trim() ?? ''
      const label = await links.nth(i).getAttribute('aria-label')
      expect(name.length > 0 || (label ?? '').length > 0, `link ${i} has no accessible name`).toBe(true)
    }

    // Each section is a landmark with a heading rather than an unlabelled block of numbers.
    for (const testid of ['collection-summary', 'status-breakdown', 'breakdown-series', 'breakdown-manufacturer']) {
      await expect(page.getByTestId(testid)).toHaveAttribute('aria-labelledby', /.+/)
    }

    // And a status figure can actually be operated from the keyboard, not merely focused.
    await page.getByTestId('status-owned').focus()
    await expect(page.getByTestId('status-owned')).toBeFocused()
    await page.keyboard.press('Enter')
    await page.waitForURL(/status=owned/)
  })

  test('the remainder is not presented as something to follow', async ({ page }) => {
    // Nine distinct series exceeds the bound of eight, so a remainder group appears. It has no
    // filter that reproduces it, and offering it as a link would promise a gallery that cannot be
    // built.
    const stamp = Date.now()
    for (let i = 0; i < 9; i++) {
      await seedCollectible(page, {
        name: `Rem ${i} ${stamp}`, status: 'owned', price: '1.00', series: `Series ${i}`,
      })
    }
    await gotoReady(page, '/collection/stats')

    const remainder = page.getByTestId('breakdown-series').getByText(/everything else/i).first()
    await expect(remainder).toBeVisible()
    await expect(
      page.getByTestId('breakdown-series').getByRole('link', { name: /everything else/i }),
    ).toHaveCount(0)
  })
})
