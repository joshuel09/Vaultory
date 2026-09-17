import { expect, type Page } from '@playwright/test'

/**
 * These journeys need the whole stack running: PostgreSQL with migrations applied, the Go service,
 * and the Next.js frontend. See specs/001-add-browse-collectibles/quickstart.md.
 */

/** Sign in as a seeded development collector. "second" selects the other one, for privacy checks. */
export async function signIn(page: Page, collector: 'primary' | 'second' = 'primary') {
  const url = collector === 'second' ? '/api/dev/session?collector=second' : '/api/dev/session'
  const response = await page.request.post(url)
  expect(response.ok(), 'the development sign-in endpoint must be available').toBeTruthy()
}

/**
 * Navigate, and wait until the page is actually interactive.
 *
 * Not ceremony. These pages are server-rendered and their inputs are React-controlled, so text
 * typed before hydration lives only in the DOM: React's state never sees it, and the next render
 * throws it away. A human rarely types that fast; Playwright always does. Without this wait the
 * name field is silently emptied by the next interaction and the form submits blank, which the
 * server then rejects as "a name is required" — a failure that looks nothing like its cause.
 */
export async function gotoReady(page: Page, path: string) {
  await page.goto(path)
  await page.waitForLoadState('networkidle')
}

/** Add a collectible through the form, as a collector would. */
export async function addCollectible(
  page: Page,
  name: string,
  status: 'owned' | 'preordered' | 'wishlist' | 'sold' = 'owned',
) {
  await gotoReady(page, '/collection/new')
  await page.getByLabel(/^name/i).fill(name)
  await page.getByLabel(/collection status/i).selectOption(status)
  // Proves the value survived the status change. If hydration were still pending this would fail
  // here, naming the real problem, instead of submitting an empty form and blaming validation.
  await expect(page.getByLabel(/^name/i)).toHaveValue(name)
  await page.getByRole('button', { name: /add to my vault/i }).click()
  await expect(page.getByTestId('add-success')).toBeVisible()
  // A successful save calls router.refresh(), which re-navigates this route. Leaving before that
  // settles makes the next page.goto abort with "interrupted by another navigation" — pointing at
  // whichever test navigated next rather than at the add that is still finishing.
  await page.waitForLoadState('networkidle')
}

/** A PNG of the given size, built in the browser, for upload tests. */
export const oversizedFile = {
  name: 'too-big.jpg',
  mimeType: 'image/jpeg',
  // Just over 10 MB. Not a real image, which is fine: the size check comes first (FR-010).
  buffer: Buffer.alloc(10 * 1024 * 1024 + 1024, 0x5a),
}

export const notAnImage = {
  name: 'document.jpg',
  mimeType: 'image/jpeg',
  // A PDF wearing an image name and an image type. Only decoding catches this (FR-009).
  buffer: Buffer.from('%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\n'),
}
