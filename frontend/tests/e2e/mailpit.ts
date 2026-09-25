import type { APIRequestContext } from '@playwright/test'

/**
 * Reading the mail Vaultory actually sent.
 *
 * Mailpit is reachable inside the Compose network as `mailpit` and from the host as localhost, so
 * the base URL follows whichever the suite is using.
 */
const MAILPIT = process.env.MAILPIT_URL ?? 'http://mailpit:8025'

export async function clearMailbox(request: APIRequestContext): Promise<void> {
  await request.delete(`${MAILPIT}/api/v1/messages`)
}

/** The newest message sent to an address, waiting briefly for it to arrive. */
export async function latestMessageTo(
  request: APIRequestContext,
  email: string,
  timeoutMs = 10_000,
): Promise<{ subject: string; text: string }> {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    const list = await request.get(`${MAILPIT}/api/v1/search?query=${encodeURIComponent(`to:${email}`)}`)
    if (list.ok()) {
      const body = (await list.json()) as { messages?: { ID: string; Subject: string }[] }
      const first = body.messages?.[0]
      if (first) {
        const full = await request.get(`${MAILPIT}/api/v1/message/${first.ID}`)
        const detail = (await full.json()) as { Text?: string }
        return { subject: first.Subject, text: detail.Text ?? '' }
      }
    }
    await new Promise((r) => setTimeout(r, 400))
  }
  throw new Error(
    `No message reached ${email} within ${timeoutMs}ms. A send that goes nowhere looks exactly ` +
      'like one that worked, which is why this fails loudly rather than returning empty.',
  )
}

/**
 * The first link in a message body, exactly as it was sent.
 *
 * No rewriting: the e2e container shares the frontend's network namespace, so localhost is the app
 * here too and the suite follows precisely the link a collector receives. An earlier version
 * rewrote the origin, which worked and quietly hid the fact that Better Auth's own redirects point
 * at the configured origin as well — those cannot be rewritten from a test.
 */
export function linkIn(text: string): string {
  const match = /https?:\/\/\S+/.exec(text)
  if (!match) throw new Error(`No link found in the message:\n${text.slice(0, 400)}`)
  return match[0]
}
