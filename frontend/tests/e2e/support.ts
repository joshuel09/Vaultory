import { expect, type Page } from '@playwright/test'

/**
 * These journeys need the whole stack running: PostgreSQL with migrations applied, the Go service,
 * and the Next.js frontend. See specs/001-add-browse-collectibles/quickstart.md.
 */

/**
 * Sign in by creating a real account.
 *
 * Feature 004 deleted the development sign-in this used to call, which is the point of that
 * feature: there is no way into a vault except registering. Each call makes a fresh account, so
 * tests cannot see one another's collectibles and "second" is genuinely a different collector
 * rather than a fixture chosen by a query parameter.
 */
export async function signIn(page: Page, collector: 'primary' | 'second' = 'primary') {
  const email = `e2e-${collector}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}@example.test`
  /*
   * The Origin header is set by hand because page.request does not send one from about:blank, and
   * Better Auth refuses a state-changing request without it — MISSING_OR_NULL_ORIGIN, which is a
   * CSRF defence doing its job rather than a bug. A real browser always sends one; this helper
   * runs before any navigation, so it has to say so itself.
   */
  const origin = process.env.VAULTORY_BASE_URL ?? 'http://localhost:3000'
  const response = await page.request.post('/api/auth/sign-up/email', {
    headers: { Origin: origin },
    data: { email, password: 'a-long-enough-password', name: email.split('@')[0] },
  })
  if (!response.ok()) {
    // The body is the only thing that distinguishes an origin rejection from a rate limit from a
    // duplicate address, and guessing between them wastes more time than printing it.
    const body = await response.text().catch(() => '<unreadable>')
    throw new Error(
      `registration failed with ${response.status()}: ${body.slice(0, 300)}\n` +
        'A 403 is usually BETTER_AUTH_TRUSTED_ORIGINS; a 429 is the sign-up rate limit.',
    )
  }
  return email
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

/**
 * A real 320x400 JPEG, so the backend's decode-and-render path runs for real.
 *
 * Generated rather than hand-written: the format check decodes the bytes rather than trusting the
 * declared type (FR-009), so a stub with a .jpg name would be refused exactly as `notAnImage` is.
 * Two of them, so a replacement is distinguishable from what it replaced.
 */
function jpegFrom(base64: string, name: string) {
  return { name, mimeType: 'image/jpeg', buffer: Buffer.from(base64, 'base64') }
}

const JPEG_BASE64 =
  '/9j/2wCEAAoHBwgHBgoICAgLCgoLDhgQDg0NDh0VFhEYIx8lJCIfIiEmKzcvJik0KSEiMEExNDk7Pj4+JS5ESUM8SDc9PjsBCgsLDg0OHBAQHDsoIig7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7Ozs7O//AABEIAZABQAMBIgACEQEDEQH/xAGiAAABBQEBAQEBAQAAAAAAAAAAAQIDBAUGBwgJCgsQAAIBAwMCBAMFBQQEAAABfQECAwAEEQUSITFBBhNRYQcicRQygZGhCCNCscEVUtHwJDNicoIJChYXGBkaJSYnKCkqNDU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6g4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2drh4uPk5ebn6Onq8fLz9PX29/j5+gEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoLEQACAQIEBAMEBwUEBAABAncAAQIDEQQFITEGEkFRB2FxEyIygQgUQpGhscEJIzNS8BVictEKFiQ04SXxFxgZGiYnKCkqNTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqCg4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2dri4+Tl5ufo6ery8/T19vf4+fr/2gAMAwEAAhEDEQA/APKgtOC0oWnBa+9bOWLGhacFpwWnBals2ixoWnBacFpwWpbN4saFpwWnBacFqWzaLGhacFpwWnBahs3ixoWnBacFpQtS2bRYgWnBaULTgtS2bxY0LTgtOC04LUtm0WNC04LTgtOC1LZtFjQtOC04LTgtS2bxY0LTgtOC04LUtm0WNC04LTgtKFqGzeLEC04LShacFqWzaLGhacFpwWnBals2ixoWnBacFpwWpbN4saFpwWnBacFqWzaLOAC04LShacFr6Bs/H4saFpwWnBacFqWzaLGhacFpwWnBals3ixoWnBacFpwWpbNosqhacFpwWnBaps8eLGhacFpwWnBals2ixoWnBacFpQtQ2bxYgWnBaULTgtS2bRY0LTgtOC04LUtm0WNC04LTgtOC1LZvFjQtOC04LTgtS2bRY0LTgtOC04LUtm8WNC04LTgtKFqWzaLEC04LShacFqGzaLGhacFpwWnBals3ixoWnBacFpwWpbNosaFpwWnBacFqWzeLGhacFpwWnBals2ixoWnBacFpQtS2bRYgWnBaULTgtQ2bxZwAWnBacFpwWvoWz8fixoWnBacFpwWpbNosaFpwWnBaULUNm8WIFpwWlC04LUtm0WVgtOC0oWnBats8aLEC0oWnBacFqWzeLGhacFpwWnBahs2ixoWnBacFpwWpbN4saFpwWnBacFqWzaLGhacFpwWnBals2ixoWnBaULTgtS2bxY0LTgtOC04LUNm0WNC04LTgtOC1LZvFjQtOC04LTgtS2bRY0LTgtOC04LUtm0WNC04LTgtKFqWzeLEC04LShacFqWzaLGhacFpwWnBahs3ixoWnBacFpwWpbNosaFpwWnBacFqWzaLOAC04LShacFr6Fs/H4sQLShacFpwWpbN4saFpwWnBacFqGzaLGhacFpwWnBals3iyqFpwWnBacFqmzxosaFpwWnBacFqWzaLGhacFpwWnBals3ixoWnBaULTgtS2bRYgWlC04LTgtS2bxY0LTgtOC04LUNm0WNC04LTgtOC1LZtFjQtOC04LTgtS2bxY0LTgtOC0oWpbNosQLTgtKFpwWpbN4saFpwWnBacFqWzaLGhacFpwWnBals2ixoWnBacFpwWpbN4saFpwWnBacFqGzaLGhacFpwWlC1LZvFiBacFpQtOC1LZtFnABacFpwWnBa+gbPx+LGhacFpwWnBals2ixoWnBacFpwWpbN4saFpwWlC04LUtm0WVgtOC0oWnBaps8eLEC0oWnBacFqWzaLGhacFpwWnBals2ixoWnBacFpwWpbN4saFpwWnBacFqGzaLGhacFpwWnBals3ixoWnBaULTgtS2bRY0LTgtOC04LUtm0WNC04LTgtOC1LZvFjQtOC04LTgtS2bRY0LTgtOC04LUtm8WNC04LTgtKFqGzaLEC04LShacFqWzaLGhacFpwWnBals3ixoWnBacFpwWpbNosaFpwWnBacFqWzeLOAC04LShacFr6Bs/H4sQLShacFpwWpbNosaFpwWnBacFqWzaLGhacFpwWnBals3iyqFpwWnBacFqmzxosaFpwWnBacFqWzeLGhacFpwWnBals2ixoWnBaULTgtQ2bxYgWlC04LTgtS2bRY0LTgtOC04LUtm0WNC04LTgtOC1LZvFjQtOC04LTgtS2bRY0LTgtOC04LUtm8WNC04LShacFqGzaLEC0oWnBacFqWzaLGhacFpwWnBals3ixoWnBacFpwWpbNosaFpwWnBacFqWzeLGhacFpwWlC1LZtFiBacFpQtOC1LZtFnABacFpwWnBa+gbPx+LGhacFpwWnBals3ixoWnBacFpwWpbNosaFpwWlC04LUNm8WVgtOC0oWnBats8aLEC0oWnBacFqGzaLGhacFpwWnBals3ixoWnBacFpwWpbNosaFpwWnBacFqWzeLGhacFpwWnBals2ixoWnBaULTgtS2bRYgWlC04LTgtQ2bxY0LTgtOC04LUtm0WNC04LTgtOC1LZvFjQtOC04LTgtS2bRY0LTgtOC04LUtm0WNC04LShacFqWzeLEC0oWnBacFqWzaLGhacFpwWnBahs3ixoWnBacFpwWpbNos4ALTgtKFpwWvoWz8fixAtKFpwWnBahs2ixoWnBacFpwWpbN4saFpwWnBacFqWzaLKoWnBacFpwWqbPHixoWnBacFpwWpbNosaFpwWnBacFqWzaLGhacFpwWlC1LZvFiBaULTgtOC1LZtFjQtOC04LTgtS2bxY0LTgtOC04LUtm0WNC04LTgtOC1DZtFjQtOC04LTgtS2bxY0LTgtKFpwWpbNosQLShacFpwWpbN4saFpwWnBacFqWzaLGhacFpwWnBahs2ixoWnBacFpwWpbN4saFpwWnBacFqWzaLGhacFpQtOC1LZvFnABacFpwWnBa+gbPx+LGhacFpwWnBals2ixoWnBacFpwWpbNosaFpwWnBaULUtm8WVgtOC04LShaps8aLEC04LShacFqWzeLGhacFpwWnBals2ixoWnBacFpwWpbNosaFpwWnBacFqWzeLGhacFpwWnBahs2ixoWnBacFpQtS2bxYgWlC04LTgtS2bRY0LTgtOC04LUtm0WNC04LTgtOC1LZvFjQtOC04LTgtQ2bRY0LTgtOC04LUtm8WNC04LShacFqWzaLEC0oWnBacFqWzaLGhacFpwWnBals3ixoWnBacFpwWpbNos4ALTgtOC0oWvoGz8fixAtOC0oWnBals3ixoWnBacFpwWpbNosaFpwWnBacFqWzaLKoWnBacFpwWqbPHixoWnBacFpwWpbNosaFpwWnBacFqWzeLGhacFpwWlC1DZtFiBaULTgtOC1LZtFjQtOC04LTgtS2bxY0LTgtOC04LUtm0WNC04LTgtOC1LZvFjQtOC04LTgtS2bRY0LTgtOC0oWobNosQLShacFpwWpbN4saFpwWnBacFqWzaLGhacFpwWnBals3ixoWnBacFpwWpbNosaFpwWnBacFqWzeLGhacFpQtOC1LZtFnABacFpwWnBa+gbPx+LGhacFpwWnBals2ixoWnBacFpwWpbN4saFpwWnBaULUNm0WVgtOC04LShats8aLEC04LShacFqGzeLGhacFpwWnBals2ixoWnBacFpwWpbN4saFpwWnBacFqWzaLGhacFpwWnBals2ixoWnBacFpQtS2bxYgWnBaULTgtS2bRY0LTgtOC04LUNm8WNC04LTgtOC1LZtFjQtOC04LTgtS2bxY0LTgtOC04LUtm0WNC04LShacFqWzaLEC0oWnBacFqWzeLGhacFpwWnBahs2ixoWnBacFpwWpbN4s4ALTgtOC0oWvoWz8eixAtOC0oWnBahs3ixoWnBacFpwWpbNosaFpwWnBacFqWzeLKoWnBacFpwWrbPGixoWnBacFpwWobN4saFpwWnBacFqWzaLGhacFpwWlC1LZtFiBacFpQtOC1LZvFjQtOC04LTgtS2bRY0LTgtOC04LUNm8WNC04LTgtOC1LZtFjQtOC04LTgtS2bRY0LTgtOC0oWpbN4sQLTgtKFpwWpbNosaFpwWnBacFqWzeLGhacFpwWnBahs2ixoWnBacFpwWpbNosaFpwWnBacFqWzeLGhacFpQtOC1LZtFnABacFpwWnBa+hbPx+LGhacFpwWnBahs3ixoWnBacFpwWpbNosaFpwWnBaULUtm0WVgtOC04LTgtU2ePFjQtOC0oWnBals2ixAtKFpwWnBals3ixoWnBacFpwWpbNosaFpwWnBacFqGzaLGhacFpwWnBals3ixoWnBacFpQtS2bRYgWnBaULTgtS2bxY0LTgtOC04LUtm0WNC04LTgtOC1LZtFjQtOC04LTgtS2bxY0LTgtOC04LUtm0WNC04LTgtKFqGzeLEC04LShacFqWzaLGhacFpwWnBals2ixoWnBacFpwWpbN4s4ALTgtOC04LX0DZ+PxY0LTgtKFpwWpbNosQLShacFpwWpbN4saFpwW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'

export const firstPhoto = jpegFrom(JPEG_BASE64, 'first-photo.jpg')
export const replacementPhoto = jpegFrom(JPEG_BASE64, 'replacement-photo.jpg')

/** Open a collectible's edit screen from its card in the gallery. */
export async function openEditScreen(page: import('@playwright/test').Page, name: string) {
  await gotoReady(page, '/collection')
  const card = page.getByTestId('collectible-card').filter({ hasText: name }).first()
  await card.getByTestId('edit-collectible').click()
  await page.waitForURL(/\/collection\/[^/]+\/edit/)
  await page.waitForLoadState('networkidle')
}
