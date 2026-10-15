import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { GroupBreakdown } from '@/components/collection/GroupBreakdown'
import { filterForGroup, hrefForGroup } from '@/lib/api/statistics-links'
import { galleryHref } from '@/lib/gallery-link'
import type { GroupSummary } from '@/lib/api/types'

function group(partial: Partial<GroupSummary> = {}): GroupSummary {
  return {
    label: 'Fate',
    kind: 'named',
    count: 3,
    pricedCount: 3,
    total: '30.00',
    ...partial,
  } as GroupSummary
}

describe('GroupBreakdown', () => {
  // T061 (FR-022): the two special groups are recognised by `kind`, never by what they are called.
  //
  // The fixture is the adversarial one: a collector whose manufacturer really is spelled "Not
  // specified", alongside the actual unspecified group. A component matching on the label would
  // treat the first as special and the second as ordinary — exactly backwards.
  it('decides a group is special from its kind, not from its label', () => {
    const groups = [
      group({ label: 'Not specified', kind: 'named', count: 5 }),
      group({ label: 'Not specified', kind: 'unspecified', count: 2 }),
      group({ label: 'Everything else', kind: 'remainder', count: 1 }),
    ]
    render(
      <GroupBreakdown
        dimension="manufacturer"
        groups={groups}
        heading="Manufacturers"
        emptyNote="none"
      />,
    )

    // The collector who typed those words gets a working link to their own pieces.
    expect(screen.getByTestId('group-manufacturer-0')).toHaveAttribute(
      'href',
      '/collection?manufacturer=Not+specified',
    )
    // The genuinely unspecified group gets the empty filter, which is a different request.
    expect(screen.getByTestId('group-manufacturer-1')).toHaveAttribute(
      'href',
      '/collection?manufacturer=',
    )
    // And the remainder is not a link at all, whatever it is labelled.
    expect(screen.getByTestId('group-manufacturer-2')).not.toHaveAttribute('href')
  })

  it('renders every group the backend sent, in the order it sent them', () => {
    const groups = [
      group({ label: 'Fate', count: 9 }),
      group({ label: 'Re:Zero', count: 4 }),
      group({ label: 'Not specified', kind: 'unspecified', count: 40 }),
    ]
    render(
      <GroupBreakdown dimension="series" groups={groups} heading="Series" emptyNote="none" />,
    )

    // 40 against 9: the unspecified group is by far the largest, and stays last because the backend
    // put it last. Reordering here would overrule FR-022 from the presentation layer.
    expect(screen.getByTestId('group-series-0-label')).toHaveTextContent('Fate')
    expect(screen.getByTestId('group-series-1-label')).toHaveTextContent('Re:Zero')
    expect(screen.getByTestId('group-series-2-count')).toHaveTextContent('40')
  })

  it('shows the counts and totals it was given', () => {
    render(
      <GroupBreakdown
        dimension="series"
        groups={[group({ label: 'Fate', count: 7, total: '1234.50' })]}
        heading="Series"
        emptyNote="none"
      />,
    )
    expect(screen.getByTestId('group-series-0-count')).toHaveTextContent('7')
    expect(screen.getByTestId('group-series-0-total')).toHaveTextContent('1,234.50')
  })

  it('says so plainly when there are no groups at all', () => {
    render(
      <GroupBreakdown
        dimension="series"
        groups={[]}
        heading="Series"
        emptyNote="None of your pieces records a series yet."
      />,
    )
    expect(screen.getByTestId('breakdown-series')).toHaveTextContent(
      'None of your pieces records a series yet.',
    )
  })
})

// T062 (FR-031, SC-010): the link a group carries is the filter that reproduces that group.
//
// The backend asserts the other half of this — that the filter returns exactly as many collectibles
// as the group claimed (TestEachGroupsCountMatchesItsOwnLink). These two together are what make the
// number traceable; either alone leaves a gap where the two sides disagree.
describe('the filter behind a group', () => {
  it('filters a named group on its own label', () => {
    expect(filterForGroup('series', group({ label: 'Re:Zero' }))).toEqual({ series: 'Re:Zero' })
    expect(hrefForGroup('series', group({ label: 'Re:Zero' }))).toBe('/collection?series=Re%3AZero')
  })

  it('filters the unspecified group on an empty value, not on no filter', () => {
    const filter = filterForGroup('manufacturer', group({ kind: 'unspecified', label: 'x' }))
    expect(filter).toEqual({ manufacturer: '' })
    // The distinction this whole seam turns on: '' is a filter, absent is not.
    expect(galleryHref(filter!)).toBe('/collection?manufacturer=')
    expect(galleryHref({})).toBe('/collection')
  })

  it('gives the remainder no filter, because none reproduces it', () => {
    expect(filterForGroup('series', group({ kind: 'remainder' }))).toBeNull()
    expect(hrefForGroup('series', group({ kind: 'remainder' }))).toBeNull()
  })

  it('keeps an empty filter through the href builder rather than dropping it', () => {
    expect(galleryHref({ series: '' })).toBe('/collection?series=')
    expect(galleryHref({ series: null })).toBe('/collection')
    expect(galleryHref({ status: 'owned', series: '' })).toBe('/collection?status=owned&series=')
  })
})
