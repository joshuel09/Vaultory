import { describe, expect, it, vi } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { render, screen } from '@testing-library/react'
import CollectionError from '@/app/collection/error'

const source = readFileSync(resolve(__dirname, '../../app/collection/error.tsx'), 'utf8')

/**
 * FR-043 and Constitution I: the error state must be legible in both appearances.
 *
 * This replaces a browser test that provoked the boundary by dropping the session. That route no
 * longer reaches it — a missing session is a redirect now — and the test only ever asserted the
 * component rendered, never that anything was legible.
 *
 * What legibility actually rests on is using the design tokens, which are defined once per
 * appearance. A hard-coded colour is the defect, and it is checkable.
 */
describe('the collection error state', () => {
  it('tells the collector what happened and offers a way forward', () => {
    render(<CollectionError error={new Error('boom')} reset={vi.fn()} />)

    expect(screen.getByTestId('collection-error')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /try again/i })).toBeInTheDocument()
  })

  it('hard-codes no colour, so both appearances work from one definition', () => {
    const literals = source.match(/#[0-9a-f]{3,8}\b|\brgba?\(|\bhsla?\(/gi) ?? []
    expect(literals, `hard-coded colours found: ${literals.join(', ')}`).toEqual([])
  })

  it('draws only from the palette tokens', () => {
    // Every colour utility in the file must name a token rather than a Tailwind default like
    // `text-red-500`, which has no dark equivalent and would be wrong in one appearance.
    const colourClasses = source.match(/\b(?:text|bg|border)-[a-z-]+(?:\/\d+)?/g) ?? []
    const tokens = /(?:ink|surface|edge|accent|danger|success)/
    const strays = colourClasses.filter(
      (c) => !tokens.test(c) && !/^(?:text|bg|border)-(?:\[|current|transparent|inherit)/.test(c),
    )
    // Layout utilities like `border-0` are not colours; only flag ones naming a palette.
    const suspicious = strays.filter((c) => /-(?:red|green|blue|gray|grey|slate|zinc|amber|yellow)-/.test(c))
    expect(suspicious, `non-token colours: ${suspicious.join(', ')}`).toEqual([])
  })
})
