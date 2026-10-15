import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { CollectionSummary } from '@/components/collection/CollectionSummary'
import type { CollectionStatistics, StatusSummary } from '@/lib/api/types'

/**
 * The summary's whole job is to show what the backend computed, unchanged.
 *
 * These tests are written against figures that are *inconsistent on purpose* — a total that does not
 * match the count, for instance. If the component ever starts deriving a figure rather than
 * displaying it, the derived answer and the supplied one will differ and these will fail.
 */

function statuses(partial: Partial<Record<string, Partial<StatusSummary>>> = {}): StatusSummary[] {
  return (['owned', 'preordered', 'wishlist', 'sold'] as const).map((status) => ({
    status,
    count: 0,
    pricedCount: 0,
    total: '0.00',
    ...(partial[status] ?? {}),
  })) as StatusSummary[]
}

function stats(overrides: Partial<CollectionStatistics> = {}): CollectionStatistics {
  return {
    totalCount: 0,
    pricedCount: 0,
    byStatus: statuses(),
    bySeries: [],
    byManufacturer: [],
    ...overrides,
  } as CollectionStatistics
}

describe('CollectionSummary', () => {
  it('shows the figures it was given rather than any it could compute', () => {
    render(
      <CollectionSummary
        stats={stats({
          totalCount: 6,
          pricedCount: 6,
          // Deliberately not a number this component could arrive at from anything else on screen.
          byStatus: statuses({ owned: { count: 6, pricedCount: 6, total: '1234.56' } }),
        })}
      />,
    )

    expect(screen.getByTestId('total-count')).toHaveTextContent('6')
    expect(screen.getByTestId('owned-total')).toHaveTextContent('1,234.56')
  })

  it('shows owned value alone, never a sum across statuses', () => {
    render(
      <CollectionSummary
        stats={stats({
          totalCount: 4,
          pricedCount: 4,
          byStatus: statuses({
            owned: { count: 1, pricedCount: 1, total: '100.00' },
            preordered: { count: 1, pricedCount: 1, total: '200.00' },
            wishlist: { count: 1, pricedCount: 1, total: '400.00' },
            sold: { count: 1, pricedCount: 1, total: '800.00' },
          }),
        })}
      />,
    )

    expect(screen.getByTestId('owned-total')).toHaveTextContent('100.00')
    // 1500.00 is what summing all four would give. A wishlist is aspiration and a sold piece is
    // money that came back; neither belongs in what you paid for what you own (FR-012).
    expect(screen.getByTestId('owned-total')).not.toHaveTextContent('1,500.00')
  })

  it('says nothing about unpriced pieces when every piece has a price', () => {
    render(<CollectionSummary stats={stats({ totalCount: 3, pricedCount: 3 })} />)
    expect(screen.queryByTestId('unpriced-note')).not.toBeInTheDocument()
  })

  it('says how many pieces a total covers when some have no price', () => {
    render(<CollectionSummary stats={stats({ totalCount: 10, pricedCount: 4 })} />)

    const note = screen.getByTestId('unpriced-note')
    expect(note).toHaveTextContent('4 of 10')
    expect(note).toHaveTextContent('6 pieces have no price recorded')
  })

  it('uses the singular for exactly one unpriced piece', () => {
    render(<CollectionSummary stats={stats({ totalCount: 5, pricedCount: 4 })} />)
    expect(screen.getByTestId('unpriced-note')).toHaveTextContent('1 piece has no price recorded')
  })

  it('explains a zero total when nothing at all is priced', () => {
    render(<CollectionSummary stats={stats({ totalCount: 7, pricedCount: 0 })} />)

    const note = screen.getByTestId('unpriced-note')
    // A bare "0.00" over seven collectibles reads as a broken screen. FR-011 asks for a sentence
    // that tells the collector why, and what to do about it.
    expect(note).toHaveTextContent('None of your 7 pieces has a recorded price')
    expect(screen.getByTestId('owned-total')).toHaveTextContent('0.00')
  })

  it('renders an exact decimal without ever turning it into a number', () => {
    render(
      <CollectionSummary
        stats={stats({
          totalCount: 1,
          pricedCount: 1,
          // 0.1 + 0.2 territory: a value that must survive formatting unchanged.
          byStatus: statuses({ owned: { count: 1, pricedCount: 1, total: '9999999999999999.99' } }),
        })}
      />,
    )
    expect(screen.getByTestId('owned-total')).toHaveTextContent('9,999,999,999,999,999.99')
  })
})
