import { describe, expect, it } from 'vitest'
import { DEFAULT_DESTINATION, safeRedirect } from '@/lib/safe-redirect'

/**
 * FR-022a, SC-013.
 *
 * Remembering where a signed-out visitor was headed means taking a destination out of a request
 * and following it after sign-in. Unconstrained that is an open redirect, and the link genuinely
 * starts at Vaultory, which is what makes it convincing.
 */
describe('safeRedirect', () => {
  it('keeps a path inside Vaultory', () => {
    expect(safeRedirect('/collection')).toBe('/collection')
    expect(safeRedirect('/collection/new')).toBe('/collection/new')
    expect(safeRedirect('/collection?status=owned')).toBe('/collection?status=owned')
  })

  it.each([
    ['absolute https', 'https://example.com'],
    ['absolute http', 'http://example.com/collection'],
    ['protocol-relative', '//example.com'],
    ['backslash protocol-relative', '/\\example.com'],
    ['javascript scheme', 'javascript:alert(1)'],
    ['data scheme', 'data:text/html,<script>'],
    ['relative path', 'collection'],
    ['bare host', 'example.com'],
    ['empty', ''],
    ['newline smuggling', '/collection\nLocation: https://example.com'],
    ['tab', '/collection\thttps://example.com'],
  ])('refuses %s and falls back', (_name, value) => {
    expect(safeRedirect(value)).toBe(DEFAULT_DESTINATION)
  })

  it('falls back for null and undefined', () => {
    expect(safeRedirect(null)).toBe(DEFAULT_DESTINATION)
    expect(safeRedirect(undefined)).toBe(DEFAULT_DESTINATION)
  })
})
