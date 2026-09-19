/**
 * Constrain a remembered destination to Vaultory's own paths (FR-022a).
 *
 * Sending a collector somewhere after sign-in means taking a destination out of a request and
 * following it. Unconstrained, that is an open redirect: the sign-in page becomes a credible way
 * to deliver someone to a hostile site, because the link genuinely starts at Vaultory.
 *
 * Allowed: a single-slash absolute path within this site. Everything else falls back.
 */
export const DEFAULT_DESTINATION = '/collection'

// A scheme before the first slash means it is not a path at all.
const HAS_SCHEME = /^[a-z][a-z0-9+.-]*:/i

/**
 * Control characters and whitespace, which can smuggle a header or confuse a parser.
 *
 * Checked by code point rather than by regular expression: a character class containing literal
 * control characters is a lint error in its own right, and the intent reads more plainly here.
 */
function hasUnsafeCharacters(value: string): boolean {
  for (const ch of value) {
    const code = ch.codePointAt(0) ?? 0
    if (code <= 0x1f || code === 0x7f || /\s/.test(ch)) return true
  }
  return false
}

export function safeRedirect(next: string | null | undefined): string {
  if (!next) return DEFAULT_DESTINATION

  // Must be absolute within the site. "//host" and "/\host" are protocol-relative in browsers and
  // would leave the origin despite starting with a slash.
  if (!next.startsWith('/')) return DEFAULT_DESTINATION
  if (next.startsWith('//') || next.startsWith('/\\')) return DEFAULT_DESTINATION
  if (HAS_SCHEME.test(next)) return DEFAULT_DESTINATION
  if (hasUnsafeCharacters(next)) return DEFAULT_DESTINATION

  return next
}
