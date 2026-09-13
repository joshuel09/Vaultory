import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { EmptyCollection } from '@/components/collection/EmptyCollection'
import { NoResults } from '@/components/collection/NoResults'
import CollectionError from '@/app/collection/error'
import Loading from '@/app/collection/loading'

vi.mock('next/link', () => ({
  default: ({ children, href }: { children: React.ReactNode; href: string }) => (
    <a href={href}>{children}</a>
  ),
}))

describe('the collection view states', () => {
  // FR-041: an empty vault explains itself and offers the way out.
  it('the empty state offers a path to add a first collectible', () => {
    render(<EmptyCollection />)
    expect(screen.getByTestId('empty-collection')).toBeInTheDocument()
    expect(screen.getByText(/vault is empty/i)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /add a collectible/i })).toHaveAttribute(
      'href',
      '/collection/new',
    )
  })

  // FR-040: a filter matching nothing is not the same as an empty vault, and must not say it is.
  it('the no-results state is distinct from the empty state', () => {
    render(<NoResults status="sold" total={9} />)
    expect(screen.getByTestId('no-results')).toBeInTheDocument()
    expect(screen.getByText(/nothing marked sold/i)).toBeInTheDocument()
    // The decisive assertion: it must not claim the vault is empty when it holds nine.
    expect(screen.queryByText(/vault is empty/i)).not.toBeInTheDocument()
    expect(screen.getByText(/9/)).toBeInTheDocument()
  })

  it('the no-results state names the status that matched nothing', () => {
    render(<NoResults status="preordered" total={3} />)
    expect(screen.getByText(/nothing marked preordered/i)).toBeInTheDocument()
  })

  // FR-042: a loading state, and specifically not a flash of the empty state.
  it('the loading state announces itself and never claims the vault is empty', () => {
    render(<Loading />)
    expect(screen.getByRole('status')).toBeInTheDocument()
    expect(screen.getByText(/loading your collection/i)).toBeInTheDocument()
    expect(screen.queryByText(/vault is empty/i)).not.toBeInTheDocument()
    expect(screen.queryByTestId('empty-collection')).not.toBeInTheDocument()
  })

  // FR-043: an error state with a way to try again, that does not imply data loss.
  it('the error state offers a retry and does not suggest the collection is gone', () => {
    const reset = vi.fn()
    render(<CollectionError error={new Error('boom')} reset={reset} />)
    expect(screen.getByTestId('collection-error')).toBeInTheDocument()
    expect(screen.getByRole('alert')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /try again/i })).toBeInTheDocument()
    expect(screen.queryByText(/vault is empty/i)).not.toBeInTheDocument()
  })

  // Constitution IV: internal detail never reaches the collector.
  it('the error state shows no internal detail', () => {
    render(<CollectionError error={Object.assign(new Error('pgx: SELECT failed'), { digest: 'abc' })} reset={() => {}} />)
    expect(screen.queryByText(/pgx/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/SELECT/)).not.toBeInTheDocument()
  })
})
