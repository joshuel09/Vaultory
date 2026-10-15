import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { StatusBreakdown } from '@/components/collection/StatusBreakdown'
import type { StatusSummary } from '@/lib/api/types'

function statuses(
  partial: Partial<Record<string, Partial<StatusSummary>>> = {},
): StatusSummary[] {
  return (['owned', 'preordered', 'wishlist', 'sold'] as const).map((status) => ({
    status,
    count: 0,
    pricedCount: 0,
    total: '0.00',
    ...(partial[status] ?? {}),
  })) as StatusSummary[]
}

describe('StatusBreakdown', () => {
  // T045 (FR-014, FR-015): the shape of this view does not change with the vault. A collector who
  // has never preordered anything still sees Preordered, reading zero — because a status that
  // disappears when empty is indistinguishable from one the screen forgot.
  it('renders all four statuses even when three of them hold nothing', () => {
    render(<StatusBreakdown byStatus={statuses({ owned: { count: 9, pricedCount: 9, total: '42.00' } })} />)

    expect(screen.getByTestId('status-owned-count')).toHaveTextContent('9')
    for (const empty of ['preordered', 'wishlist', 'sold']) {
      expect(screen.getByTestId(`status-${empty}-count`)).toHaveTextContent('0')
      expect(screen.getByTestId(`status-${empty}-total`)).toHaveTextContent('0.00')
    }
  })

  // T046 (FR-019): wording, asserted. "Paid" is a requirement here rather than copy — nothing in
  // this product records a sale price, so any other word invites a collector to read the figure as
  // what they got back.
  it('describes the Sold total as what was paid, in those words', () => {
    render(<StatusBreakdown byStatus={statuses({ sold: { count: 2, pricedCount: 2, total: '75.50' } })} />)

    const sold = screen.getByTestId('status-sold')
    expect(sold).toHaveTextContent('75.50')
    expect(sold).toHaveTextContent('paid for these')
    expect(screen.getByTestId('sold-caveat')).toHaveTextContent(
      'Nothing here records what they sold for',
    )
  })

  // The three statuses that record no payment must not claim one. A wishlist price is what a piece
  // costs, not what anyone spent.
  it('does not call a wishlist or preorder price something that was paid', () => {
    render(<StatusBreakdown byStatus={statuses({ wishlist: { count: 1, total: '10.00' } })} />)

    expect(screen.getByTestId('status-wishlist')).not.toHaveTextContent('paid')
    expect(screen.getByTestId('status-preordered')).not.toHaveTextContent('paid')
  })

  // T041 (FR-030): every figure traces back to the pieces behind it.
  it('links each status to the gallery filtered to it', () => {
    render(<StatusBreakdown byStatus={statuses()} />)

    for (const status of ['owned', 'preordered', 'wishlist', 'sold']) {
      expect(screen.getByTestId(`status-${status}`)).toHaveAttribute(
        'href',
        `/collection?status=${status}`,
      )
    }
  })

  it('shows the totals it was given rather than any it could derive', () => {
    render(
      <StatusBreakdown
        byStatus={statuses({
          owned: { count: 1, pricedCount: 1, total: '100.00' },
          preordered: { count: 1, pricedCount: 1, total: '200.00' },
          wishlist: { count: 1, pricedCount: 1, total: '400.00' },
          sold: { count: 1, pricedCount: 1, total: '800.00' },
        })}
      />,
    )

    expect(screen.getByTestId('status-owned-total')).toHaveTextContent('100.00')
    expect(screen.getByTestId('status-preordered-total')).toHaveTextContent('200.00')
    expect(screen.getByTestId('status-wishlist-total')).toHaveTextContent('400.00')
    expect(screen.getByTestId('status-sold-total')).toHaveTextContent('800.00')
  })
})
