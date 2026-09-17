import { afterEach, describe, expect, it, vi } from 'vitest'
import { randomUUID } from '@/lib/uuid'

const V4 = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/

afterEach(() => vi.unstubAllGlobals())

describe('randomUUID', () => {
  it('uses crypto.randomUUID when it exists', () => {
    vi.stubGlobal('crypto', { ...globalThis.crypto, randomUUID: () => 'from-native' })
    expect(randomUUID()).toBe('from-native')
  })

  /**
   * The regression this file exists for. crypto.randomUUID is undefined outside a secure context —
   * plain HTTP on anything but localhost — and the add form called it during render, so the whole
   * view crashed. Every manual check used localhost and passed; the browser suite, which reaches
   * the app by container name, did not.
   */
  it('still produces a v4 UUID when crypto.randomUUID is unavailable', () => {
    vi.stubGlobal('crypto', { getRandomValues: globalThis.crypto.getRandomValues.bind(globalThis.crypto) })
    expect(typeof (globalThis.crypto as Crypto).randomUUID).not.toBe('function')

    const id = randomUUID()
    expect(id).toMatch(V4)
  })

  it('does not repeat itself', () => {
    vi.stubGlobal('crypto', { getRandomValues: globalThis.crypto.getRandomValues.bind(globalThis.crypto) })
    const ids = new Set(Array.from({ length: 500 }, () => randomUUID()))
    expect(ids.size).toBe(500)
  })
})
