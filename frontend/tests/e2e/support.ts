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

/** Add a collectible through the form, as a collector would. */
export async function addCollectible(
  page: Page,
  name: string,
  status: 'owned' | 'preordered' | 'wishlist' | 'sold' = 'owned',
) {
  await page.goto('/collection/new')
  await page.getByLabel(/^name/i).fill(name)
  await page.getByLabel(/collection status/i).selectOption(status)
  await page.getByRole('button', { name: /add to my vault/i }).click()
  await expect(page.getByTestId('add-success')).toBeVisible()
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
