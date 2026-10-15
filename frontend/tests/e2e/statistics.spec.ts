import { expect, test } from '@playwright/test'
import { addDetailedCollectible, gotoReady, signIn } from './support'

/**
 * Feature 008 end to end: what a collector sees, and whether the numbers survive the round trip.
 *
 * Each test signs in as a fresh collector, so the figures below are the whole vault rather than a
 * share of whatever earlier tests left behind. That is what makes an exact total assertable at all.
 */

test.describe('what your collection holds', () => {
  // T066 — the headline journey. Desktop only: this is about the figures, and the other two
  // viewports are covered by the responsive spec.
  test.describe('the headline', () => {
    test.skip((_, info) => info.project.name !== 'desktop', 'the figures, once, on desktop')

    test('shows what the vault holds and what was paid for the owned pieces', async ({ page }) => {
      await signIn(page)
      const stamp = Date.now()

      await addDetailedCollectible(page, {
        name: `Owned A ${stamp}`, status: 'owned', price: '1200.00', series: 'Kaiju Wars',
      })
      await addDetailedCollectible(page, {
        name: `Owned B ${stamp}`, status: 'owned', price: '49.99', series: 'Kaiju Wars',
      })
      await addDetailedCollectible(page, {
        name: `Wishlist ${stamp}`, status: 'wishlist', price: '800.00', series: 'Deep Sea',
      })

      // The named link from the gallery, rather than typing the URL: FR-029 asks for a route in,
      // and a test that navigates directly would pass with no link on the page at all.
      await gotoReady(page, '/collection')
      await page.getByTestId('statistics-link').click()
      await page.waitForURL(/\/collection\/stats/)

      await expect(page.getByTestId('total-count')).toHaveText('3')
      // 1200.00 + 49.99, and emphatically not 2049.99: a wishlist is aspiration, not spending.
      await expect(page.getByTestId('owned-total')).toHaveText('1,249.99')
      await expect(page.getByTestId('owned-total')).not.toContainText('2,049.99')

      // All four statuses, including the two holding nothing.
      for (const status of ['owned', 'preordered', 'wishlist', 'sold']) {
        await expect(page.getByTestId(`status-${status}`)).toBeVisible()
      }
      await expect(page.getByTestId('status-preordered-count')).toContainText('0')
      await expect(page.getByTestId('status-wishlist-total')).toContainText('800.00')

      // Sold says what the figure is, in the words FR-019 requires.
      await expect(page.getByTestId('sold-caveat')).toContainText(/paid/i)
      await expect(page.getByTestId('sold-caveat')).toContainText(/sold for/i)

      // And the breakdowns, with the larger series first.
      await expect(page.getByTestId('breakdown-series')).toBeVisible()
      await expect(page.getByTestId('group-series-0-label')).toHaveText('Kaiju Wars')
      await expect(page.getByTestId('group-series-0-count')).toHaveText('2')
    })

    // T067 — SC-010: a figure links to a gallery holding exactly the pieces it counted.
    //
    // The whole feature rests on this. A number a collector can follow and find disagrees with is
    // worse than no number, and nothing below this level can catch the two sides drifting apart.
    test('every figure links to exactly the pieces it counted', async ({ page }) => {
      await signIn(page)
      const stamp = Date.now()

      await addDetailedCollectible(page, { name: `Sw A ${stamp}`, status: 'owned', price: '10.00', series: 'Sword Saga' })
      await addDetailedCollectible(page, { name: `Sw B ${stamp}`, status: 'owned', price: '10.00', series: 'sword saga' })
      await addDetailedCollectible(page, { name: `Deep ${stamp}`, status: 'sold', price: '5.00', series: 'Deep Sea' })
      // No series at all: the unspecified group, whose link is the one most likely to be wrong.
      await addDetailedCollectible(page, { name: `Nameless ${stamp}`, status: 'owned', price: '7.00' })

      await gotoReady(page, '/collection/stats')

      // A status figure: three owned pieces, and three cards behind the link.
      await expect(page.getByTestId('status-owned-count')).toContainText('3')
      await page.getByTestId('status-owned').click()
      await page.waitForURL(/status=owned/)
      await expect(page.getByTestId('collectible-card')).toHaveCount(3)

      // A named series group, which folded two spellings into one.
      await gotoReady(page, '/collection/stats')
      await expect(page.getByTestId('group-series-0-label')).toHaveText(/sword saga/i)
      await expect(page.getByTestId('group-series-0-count')).toHaveText('2')
      await page.getByTestId('group-series-0').click()
      await page.waitForURL(/series=/)
      await expect(page.getByTestId('collectible-card')).toHaveCount(2)

      // And the unspecified group. Its link carries an empty value, which every shortcut on the way
      // turns into "no filter" — and then this shows four cards instead of one.
      await gotoReady(page, '/collection/stats')
      const unspecified = page.getByTestId('breakdown-series').getByRole('link', { name: /not specified/i })
      await expect(unspecified).toBeVisible()
      await unspecified.click()
      await page.waitForURL(/series=/)
      await expect(page.getByTestId('collectible-card')).toHaveCount(1)
      await expect(page.getByText(`Nameless ${stamp}`)).toBeVisible()
    })

    // T068 — FR-034, FR-011: the two states that are true but useless if shown as bare zeros.
    test('an empty vault gets a designed state, not a grid of zeros', async ({ page }) => {
      await signIn(page)

      // The link is not offered at all when there is nothing to summarise (FR-029).
      await gotoReady(page, '/collection')
      await expect(page.getByTestId('statistics-link')).toHaveCount(0)

      // And reaching the route directly says something useful rather than showing zeros.
      await gotoReady(page, '/collection/stats')
      await expect(page.getByTestId('stats-empty')).toBeVisible()
      await expect(page.getByTestId('collection-summary')).toHaveCount(0)
      await expect(page.getByRole('link', { name: /add your first collectible/i })).toBeVisible()
    })

    test('says how much of the collection a total covers when some pieces have no price', async ({ page }) => {
      await signIn(page)
      const stamp = Date.now()

      await addDetailedCollectible(page, { name: `Priced ${stamp}`, status: 'owned', price: '20.00' })
      await addDetailedCollectible(page, { name: `Unpriced ${stamp}`, status: 'owned' })

      await gotoReady(page, '/collection/stats')
      const note = page.getByTestId('unpriced-note')
      await expect(note).toBeVisible()
      await expect(note).toContainText('1 of 2')
      await expect(note).toContainText(/no price recorded/i)
    })
  })
})
