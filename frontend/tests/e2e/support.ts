import { expect, type Locator, type Page } from '@playwright/test'

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

/**
 * Fill a field, and keep filling it until the value sticks.
 *
 * React-controlled inputs discard anything typed before hydration: the text reaches the DOM, React
 * state never sees it, and the next render throws it away. `page.goto` resolves on `load`, which is
 * well before that, so a fill immediately after following a link is a race — and a lost one shows
 * up as a *validation* failure ("passwords must be at least 12 characters" over an empty field),
 * which blames the form rather than the timing.
 *
 * Retrying until the value holds is the fix rather than a longer wait, because what is being waited
 * for is an event with no deadline. Emulated mobile is slower than desktop, so this surfaced at one
 * viewport and not another and read as a viewport-specific defect (issue #41).
 */
export async function fillWhenReady(field: Locator, value: string) {
  /*
   * Wait until React owns this input before typing into it.
   *
   * React attaches `__reactProps$<hash>` to a DOM node as it hydrates it, so the marker's presence
   * is a precise signal that a fill will go through onChange rather than into a DOM that the next
   * render throws away. It beats waiting a fixed interval, which is either too short to be safe or
   * too long to be free, and it beats networkidle, which says nothing about React at all.
   *
   * The marker genuinely discriminates — it is not a no-op dressed as a fix. Measured on this app
   * by probing immediately after `goto`: absent on 3 of 12 navigations at the mobile viewport, 1 of
   * 12 at tablet, 0 of 12 at desktop. That is the same ordering as the failures this removes, which
   * is the corroboration worth having.
   *
   * If React ever stops using this marker the wait expires and the fill is attempted anyway, so the
   * worst case is the flake we already had rather than a suite that hangs.
   */
  await field.evaluate((el) =>
    new Promise<void>((resolve) => {
      const owned = () => Object.keys(el).some((k) => k.startsWith('__reactProps$'))
      if (owned()) return resolve()
      const started = Date.now()
      const timer = setInterval(() => {
        if (owned() || Date.now() - started > 10_000) {
          clearInterval(timer)
          resolve()
        }
      }, 20)
    }),
  )

  // Belt and braces: confirm the value actually took, and fill again if it did not.
  await expect(async () => {
    await field.fill(value)
    await expect(field).toHaveValue(value, { timeout: 500 })
  }).toPass({ timeout: 10_000 })
}

/** Add a collectible through the form, as a collector would. */
export async function addCollectible(
  page: Page,
  name: string,
  status: 'owned' | 'preordered' | 'wishlist' | 'sold' = 'owned',
) {
  await gotoReady(page, '/collection/new')
  // Issue #43. The first field after a navigation is the one exposed to hydration: text typed
  // before React attaches reaches the DOM and is discarded by the first render, and the form then
  // reports "a name is required" over a name that is plainly on screen. The fields after this one
  // need no gate — React owns the form by the time they are reached.
  await fillWhenReady(page.getByLabel(/^name/i), name)
  await page.getByLabel(/collection status/i).selectOption(status)
  // And the value survived the status change too.
  await expect(page.getByLabel(/^name/i)).toHaveValue(name)
  await page.getByRole('button', { name: /add to my vault/i }).click()
  await expect(page.getByTestId('add-success')).toBeVisible()
  // A successful save calls router.refresh(), which re-navigates this route. Leaving before that
  // settles makes the next page.goto abort with "interrupted by another navigation" — pointing at
  // whichever test navigated next rather than at the add that is still finishing.
  await page.waitForLoadState('networkidle')
}

/**
 * Add a collectible with the fields the statistics aggregate over.
 *
 * The plain addCollectible above fills only a name and a status, which is everything the gallery
 * needs and nothing the statistics do: with no price every total is zero, and with no series every
 * piece lands in the unspecified group. A statistics journey driven by that helper would pass
 * against a view that showed nothing at all.
 */
export async function addDetailedCollectible(
  page: Page,
  fields: {
    name: string
    status?: 'owned' | 'preordered' | 'wishlist' | 'sold'
    price?: string
    series?: string
    manufacturer?: string
  },
) {
  await gotoReady(page, '/collection/new')

  /*
   * Gate on hydration before filling anything else.
   *
   * networkidle does not mean React has attached. Text typed before it does lives only in the DOM
   * and is discarded by the next render, and this helper fills up to five fields, so it has a wider
   * window for that than the two-field one above — which is how it showed up: the name field came
   * back empty at one viewport while passing at five others in the same run.
   *
   * Retrying the fill until the value sticks is the fix rather than a longer wait, because what is
   * being waited for is an event with no deadline.
   */
  await fillWhenReady(page.getByLabel(/^name/i), fields.name)

  await page.getByLabel(/collection status/i).selectOption(fields.status ?? 'owned')
  if (fields.series !== undefined) {
    await page.getByLabel(/series or franchise/i).fill(fields.series)
  }
  if (fields.manufacturer !== undefined) {
    await page.getByLabel(/^manufacturer/i).fill(fields.manufacturer)
  }
  if (fields.price !== undefined) {
    await page.getByLabel(/purchase price/i).fill(fields.price)
  }
  await expect(page.getByLabel(/^name/i)).toHaveValue(fields.name)
  await page.getByRole('button', { name: /add to my vault/i }).click()
  await expect(page.getByTestId('add-success')).toBeVisible()
  await page.waitForLoadState('networkidle')
}

/**
 * Put a collectible straight into the vault through the API.
 *
 * For tests that need a *shape* of collection rather than the act of adding one. Nine pieces added
 * through the form is nine full page loads, nine hydrations and nine submissions, which on the
 * mobile project under three parallel workers runs past the timeout — a failure about the add form
 * in a test about the breakdown. The form path is covered by its own spec and by the journeys above.
 *
 * It goes through the same /api rewrite a browser uses, carrying the session cookie this context
 * already holds, so it is the real endpoint with the real authorization.
 */
export async function seedCollectible(
  page: Page,
  fields: {
    name: string
    status?: 'owned' | 'preordered' | 'wishlist' | 'sold'
    price?: string
    series?: string
    manufacturer?: string
  },
) {
  const response = await page.request.post('/api/collectibles', {
    data: {
      submissionKey: `e2e-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`,
      name: fields.name,
      collectionStatus: fields.status ?? 'owned',
      ...(fields.price !== undefined ? { purchasePrice: fields.price } : {}),
      ...(fields.series !== undefined ? { series: fields.series } : {}),
      ...(fields.manufacturer !== undefined ? { manufacturer: fields.manufacturer } : {}),
    },
  })
  if (!response.ok()) {
    const body = await response.text().catch(() => '<unreadable>')
    throw new Error(`seeding ${fields.name} failed with ${response.status()}: ${body.slice(0, 300)}`)
  }
}

/** A PNG of the given size, built in the browser, for upload tests. */
export const oversizedFile = {
  name: 'too-big.jpg',
  mimeType: 'image/jpeg',
  // Just over 10 MB. Not a real image, which is fine: the size check comes first (FR-010).
  buffer: Buffer.alloc(10 * 1024 * 1024 + 1024, 0x5a),
}

/**
 * A real JPEG grown to just under the 10 MB rule, for proving the largest permitted upload works.
 *
 * Padded with JPEG comment segments inserted after the SOI marker rather than with trailing bytes:
 * a decoder is entitled to stop at EOI, so junk appended to the end may or may not count toward
 * anything, while a COM segment is part of the file by definition and every decoder must carry it.
 *
 * It exists because issue #47 broke this case, not only the refusal: Next truncated a proxied body
 * at 10 MiB, which is FR-010's own limit, so a legitimate upload at the top of the range died the
 * same way an oversized one did.
 */
export function largestPermittedJpeg() {
  const raw = Buffer.from(JPEG_BASE64, 'base64')
  /*
   * Exactly MaxUploadBytes, not comfortably under it.
   *
   * This matters more than it looks, and a mutation check is what showed it: at 2 KB under the
   * limit the file plus its multipart envelope still fits inside Next's default 10 MiB cap, so the
   * test passed whether or not the bug was fixed and proved nothing. Sitting *on* the limit is what
   * pushes the envelope over the cap while staying inside what Go accepts, which is the exact
   * window issue #47 broke.
   */
  const target = 10 * 1024 * 1024
  const chunks: Buffer[] = []
  let remaining = target - raw.length
  while (remaining > 0) {
    // Each segment costs its payload plus the two length bytes; the last one is sized to land on
    // the target exactly rather than overshoot it.
    const n = remaining >= 65535 ? 65533 : remaining - 2
    const seg = Buffer.alloc(n + 2)
    seg.writeUInt16BE(0xfffe, 0)
    seg.writeUInt16BE(n, 2)
    chunks.push(seg)
    remaining -= n + 2
  }
  return {
    name: 'largest-permitted.jpg',
    mimeType: 'image/jpeg',
    buffer: Buffer.concat([raw.subarray(0, 2), ...chunks, raw.subarray(2)]),
  }
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
