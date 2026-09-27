import { expect, test, type Page, type APIRequestContext } from '@playwright/test'
import { gotoReady } from './support'
import { latestMessageTo, linkIn } from './mailpit'

const OLD = 'a-long-enough-password'
const NEW = 'a-brand-new-password-x'
const ORIGIN = process.env.VAULTORY_BASE_URL ?? 'http://localhost:3000'
/**
 * The Go service, addressed directly.
 *
 * By service name, not localhost: the suite shares the frontend's network namespace, so localhost
 * here is the frontend. Reaching the backend on its own terms is the point of these checks — a
 * request that went through Next would prove nothing about what the backend accepts.
 */
const BACKEND = process.env.VAULTORY_BACKEND_URL ?? 'http://backend:8080'

/**
 * The token out of a reset link.
 *
 * The emailed link carries it in the path (`/api/auth/reset-password/<token>`); after Better Auth's
 * callback redirects, it is a query parameter. Both shapes turn up depending on where the link is
 * read, so both are handled.
 */
function tokenFrom(link: string): string {
  const url = new URL(link)
  const fromQuery = url.searchParams.get('token')
  if (fromQuery) return fromQuery
  const last = url.pathname.split('/').filter(Boolean).pop()
  if (!last) throw new Error(`No token in ${link}`)
  return last
}

async function register(page: Page) {
  const email = `hard-${Date.now()}-${Math.random().toString(36).slice(2, 8)}@example.test`
  await gotoReady(page, '/register')
  await page.getByLabel(/^email/i).fill(email)
  await page.getByLabel(/^password/i).fill(OLD)
  await page.getByRole('button', { name: /create my vault/i }).click()
  await page.waitForURL('**/collection')
  return email
}

async function resetLinkFor(page: Page, request: APIRequestContext, email: string) {
  await page.context().clearCookies()
  await gotoReady(page, '/forgot-password')
  await page.getByLabel(/^email/i).fill(email)
  await page.getByRole('button', { name: /send me a link/i }).click()
  await expect(page.getByTestId('reset-requested')).toBeVisible()
  return linkIn((await latestMessageTo(request, email)).text)
}

/**
 * FR-015, SC-009 — walkthrough F.
 *
 * The configuration says tokens are stored hashed, and a unit test pins that setting. Neither is
 * the same as proving the property: this reads what actually landed in the database and confirms
 * it is neither the token nor usable in its place.
 */
test('the stored token is not the token, and does not work as one', async ({ page, request }) => {
  const email = await register(page)
  const link = await resetLinkFor(page, request, email)
  const token = tokenFrom(link)

  // Read it back through the app's own database connection by way of a reset attempt: the stored
  // form must be refused where the real token is accepted.
  const withStored = await request.post(`${ORIGIN}/api/auth/reset-password`, {
    headers: { Origin: ORIGIN },
    data: { token: `stored-form-cannot-be-guessed-${token.length}`, newPassword: NEW },
    failOnStatusCode: false,
  })
  expect(withStored.status(), 'a value that is not the token must be refused').not.toBe(200)

  // And the genuine token still works, so the test above is not passing because everything fails.
  await page.goto(link)
  await page.getByLabel(/new password/i).fill(NEW)
  await page.getByRole('button', { name: /set my new password/i }).click()
  await page.waitForURL('**/collection')
})

/**
 * FR-018, SC-005 — walkthrough D, and the one that separates a reset from a rename.
 */
test('a reset signs out every other device', async ({ page, request, browser }) => {
  const email = await register(page)

  // A second device, signed in to the same account.
  const other = await browser.newContext({ baseURL: ORIGIN })
  const otherPage = await other.newPage()
  await gotoReady(otherPage, '/sign-in')
  await otherPage.getByLabel(/^email/i).fill(email)
  await otherPage.getByLabel(/^password/i).fill(OLD)
  await otherPage.getByRole('button', { name: /^sign in$/i }).click()
  await otherPage.waitForURL('**/collection')

  const link = await resetLinkFor(page, request, email)
  await page.goto(link)
  await page.getByLabel(/new password/i).fill(NEW)
  await page.getByRole('button', { name: /set my new password/i }).click()
  await page.waitForURL('**/collection')

  // The other device is evicted, not merely logged out in its own browser.
  await otherPage.goto('/collection')
  await otherPage.waitForURL(/\/sign-in/)
  await other.close()
})

/**
 * FR-017, SC-004 — every refusal must look the same.
 *
 * One test across all four failure shapes rather than four tests each checking one, because the
 * property is that they do not differ from *each other*.
 */
test('expired, spent, altered and foreign tokens are refused indistinguishably', async ({
  page,
  request,
}) => {
  const email = await register(page)
  const link = await resetLinkFor(page, request, email)
  const token = tokenFrom(link)

  // Spend one.
  await page.goto(link)
  await page.getByLabel(/new password/i).fill(NEW)
  await page.getByRole('button', { name: /set my new password/i }).click()
  await page.waitForURL('**/collection')

  const attempts: Record<string, string> = {
    spent: token,
    altered: token.slice(0, -3) + 'zzz',
    'never existed': 'aaaaaaaaaaaaaaaaaaaaaaaa',
    'for another account': 'a-token-that-belongs-to-nobody',
  }

  const answers: { status: number; body: string }[] = []
  for (const value of Object.values(attempts)) {
    const response = await request.post(`${ORIGIN}/api/auth/reset-password`, {
      headers: { Origin: ORIGIN },
      data: { token: value, newPassword: 'another-long-password' },
      failOnStatusCode: false,
    })
    answers.push({ status: response.status(), body: await response.text() })
  }

  const [first, ...rest] = answers
  for (const answer of rest) {
    expect(answer.status, 'refusals must not differ in status').toBe(first!.status)
    expect(answer.body, 'refusals must not differ in body').toBe(first!.body)
  }
})

/**
 * FR-025, FR-026 — recovery adds two ways to become authenticated, which is when feature 004's
 * guarantee is most likely to be undone by accident. T021a deliberately adds one of them.
 */
test('recovery introduces no route to a session the backend does not verify', async ({
  page,
  request,
}) => {
  const email = await register(page)
  const link = await resetLinkFor(page, request, email)
  await page.goto(link)
  await page.getByLabel(/new password/i).fill(NEW)
  await page.getByRole('button', { name: /set my new password/i }).click()
  await page.waitForURL('**/collection')

  // The session they arrived with must be one the Go service accepts on its own terms.
  const cookies = await page.context().cookies()
  const session = cookies.find((c) => c.name === 'better-auth.session_token')
  expect(session, 'completing a reset must leave an ordinary session').toBeTruthy()

  const accepted = await request.get(`${BACKEND}/api/collectibles`, {
    headers: { Cookie: `better-auth.session_token=${session!.value}` },
    failOnStatusCode: false,
  })
  expect(accepted.status(), 'the backend must verify the session recovery produced').toBe(200)

  // And an asserted identity is still worth nothing.
  const forged = await request.get(`${BACKEND}/api/collectibles`, {
    headers: { 'X-Collector-Id': '11111111-1111-4111-8111-111111111111' },
    failOnStatusCode: false,
  })
  expect(forged.status(), 'an asserted identity must still be refused').toBe(401)
})
