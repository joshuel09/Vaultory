import { expect, test } from '@playwright/test'
import { addCollectible, gotoReady, openEditScreen, signIn } from './support'

test.describe('editing a collectible', () => {
  test.beforeEach(async ({ page }) => signIn(page))

  // User Story 1's independent test: correct a collectible, and confirm it has not moved.
  test('corrects a collectible without moving it in the gallery', async ({ page }) => {
    const first = `Kaiju Sentinel ${Date.now()}`
    const second = `Harbour Golem ${Date.now()}`
    await addCollectible(page, first, 'preordered')
    // Added afterwards, so `first` is the older of the two and sits second in a newest-first
    // gallery. A collectible jumping to the front after an edit would be unmistakable.
    await addCollectible(page, second)

    await openEditScreen(page, first)

    // FR-002: every stored value is already there.
    await expect(page.getByLabel(/^name/i)).toHaveValue(first)
    await expect(page.getByLabel(/collection status/i)).toHaveValue('preordered')
    // FR-003: what was never supplied is empty, not defaulted.
    await expect(page.getByLabel(/manufacturer/i)).toHaveValue('')

    const corrected = `${first} MkII`
    await page.getByLabel(/^name/i).fill(corrected)
    await page.getByLabel(/collection status/i).selectOption('owned')
    await page.getByRole('button', { name: /save changes/i }).click()

    await page.waitForURL(/\/collection(\?|$)/)
    await expect(page.getByText(corrected)).toBeVisible()

    // Reload before judging anything: a client-side update that looks right and a stored change
    // are different things.
    await gotoReady(page, '/collection')
    const cards = page.getByTestId('collectible-card')
    await expect(cards).toHaveCount(2)
    // FR-028: still second. Newest first, and editing writes nothing that affects the order.
    await expect(cards.nth(0)).toContainText(second)
    await expect(cards.nth(1)).toContainText(corrected)
    // FR-008: editing never adds.
    await expect(page.getByText(first, { exact: true })).toHaveCount(0)
  })

  test('clears an optional value rather than zeroing it', async ({ page }) => {
    const name = `Priced ${Date.now()}`
    await gotoReady(page, '/collection/new')
    await page.getByLabel(/^name/i).fill(name)
    await page.getByLabel(/purchase price/i).fill('1250.00')
    await page.getByRole('button', { name: /add to my vault/i }).click()
    await expect(page.getByTestId('add-success')).toBeVisible()
    await page.waitForLoadState('networkidle')

    await openEditScreen(page, name)
    await expect(page.getByLabel(/purchase price/i)).toHaveValue('1250.00')
    await page.getByLabel(/purchase price/i).fill('')
    await page.getByRole('button', { name: /save changes/i }).click()
    await page.waitForURL(/\/collection(\?|$)/)

    await openEditScreen(page, name)
    // Empty, not "0.00" — "not recorded" is not the same as a recorded zero (FR-005).
    await expect(page.getByLabel(/purchase price/i)).toHaveValue('')
  })

  test('keeps what was typed when the save is refused', async ({ page }) => {
    const name = `Refused ${Date.now()}`
    await addCollectible(page, name)
    await openEditScreen(page, name)

    await page.getByLabel(/purchase price/i).fill('-5.00')
    await page.getByLabel(/notes/i).fill('Still here afterwards')
    await page.getByRole('button', { name: /save changes/i }).click()

    await expect(page.getByText(/cannot be negative/i)).toBeVisible()
    // Losing a filled-in form to a failed save is the thing this must never do (FR-035).
    await expect(page.getByLabel(/notes/i)).toHaveValue('Still here afterwards')
    await expect(page.getByLabel(/purchase price/i)).toHaveValue('-5.00')
  })

  test('refuses a save made from a stale view and shows what it now says', async ({ page, context }) => {
    const name = `Contested ${Date.now()}`
    await addCollectible(page, name)
    await openEditScreen(page, name)
    const staleUrl = page.url()

    // A second tab edits first, exactly as two devices would.
    const other = await context.newPage()
    await other.goto(staleUrl)
    await other.waitForLoadState('networkidle')
    await other.getByLabel(/^name/i).fill(`${name} won the race`)
    await other.getByRole('button', { name: /save changes/i }).click()
    await other.waitForURL(/\/collection(\?|$)/)
    await other.close()

    // The first tab saves from the version it was opened at. Deliberately no reload — a reload
    // would hand it the current version and the test would pass for the wrong reason.
    await page.getByLabel(/notes/i).fill('Written in the stale tab')
    await page.getByRole('button', { name: /save changes/i }).click()

    const notice = page.getByTestId('version-conflict')
    await expect(notice).toBeVisible()
    // Not merely "it changed" — the collector has to see how, or they have nothing to decide on.
    await expect(notice).toContainText(`${name} won the race`)
    // Their own work is still in front of them, and they are still on the edit screen.
    await expect(page.getByLabel(/notes/i)).toHaveValue('Written in the stale tab')

    // Adopting the current values is a button they press, never something done to them.
    await page.getByRole('button', { name: /use these values/i }).click()
    await expect(page.getByLabel(/^name/i)).toHaveValue(`${name} won the race`)
    await expect(notice).toBeHidden()

    // And now the save lands.
    await page.getByRole('button', { name: /save changes/i }).click()
    await page.waitForURL(/\/collection(\?|$)/)
  })

  test('another collector cannot reach the edit screen', async ({ page, context }) => {
    const name = `Private ${Date.now()}`
    await addCollectible(page, name)
    await openEditScreen(page, name)
    const theirUrl = page.url()

    const intruder = await context.newPage()
    await signIn(intruder, 'second')
    await intruder.goto(theirUrl)
    // Not found, and nothing that distinguishes it from an id that never existed (FR-031).
    await expect(intruder.getByText(/not in your vault/i)).toBeVisible()
    await expect(intruder.getByLabel(/^name/i)).toHaveCount(0)
    await intruder.close()
  })

  test('returns to the filtered gallery the collector came from', async ({ page }) => {
    const name = `Filtered ${Date.now()}`
    await addCollectible(page, name, 'sold')

    await gotoReady(page, '/collection?status=sold')
    await page.getByTestId('collectible-card').filter({ hasText: name }).first()
      .getByTestId('edit-collectible').click()
    await page.waitForURL(/\/collection\/[^/]+\/edit/)
    await page.waitForLoadState('networkidle')

    await page.getByLabel(/notes/i).fill('Still sold')
    await page.getByRole('button', { name: /save changes/i }).click()

    // Back where they were browsing, not dropped on an unfiltered gallery.
    await page.waitForURL(/\/collection\?status=sold/)
    await expect(page.getByText(name)).toBeVisible()
  })
})
