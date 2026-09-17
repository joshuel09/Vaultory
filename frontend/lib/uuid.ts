/**
 * A v4 UUID that works outside a secure context.
 *
 * `crypto.randomUUID()` is only defined on HTTPS and on localhost. On a plain-HTTP origin that is
 * not localhost — a LAN address like http://192.168.1.5:3000, a container reached by service name,
 * or any deployment behind TLS termination that talks HTTP internally — it is `undefined`, and
 * calling it throws. `crypto.getRandomValues()` has no such restriction, so it is what the
 * fallback is built from: still cryptographically random, just assembled by hand.
 *
 * This is used for the submission key that makes adding a collectible idempotent, so it has to be
 * unique but is never a secret.
 */
export function randomUUID(): string {
  if (typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }

  const bytes = new Uint8Array(16)
  crypto.getRandomValues(bytes)
  // Version 4, variant 1, per RFC 4122.
  bytes[6] = ((bytes[6] ?? 0) & 0x0f) | 0x40
  bytes[8] = ((bytes[8] ?? 0) & 0x3f) | 0x80

  const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('')
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
}
