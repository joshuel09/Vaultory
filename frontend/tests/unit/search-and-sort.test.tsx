import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { SearchAndSort } from '@/components/collection/SearchAndSort'
import { NoResults } from '@/components/collection/NoResults'
import { galleryHref, isFiltered } from '@/lib/gallery-link'

// Feature 011: searching and ordering the gallery.

const push = vi.fn()
let current = new URLSearchParams('')
vi.mock('next/navigation', () => ({
  useRouter: () => ({ push }),
  usePathname: () => '/collection',
  useSearchParams: () => current,
}))

beforeEach(() => {
  push.mockClear()
  current = new URLSearchParams('')
})

describe('galleryHref with search and sort (FR-011)', () => {
  it('writes a search and a non-default order beside the filters', () => {
    expect(galleryHref({ status: 'owned', q: ' kaiju prime ', sort: 'price_desc' })).toBe(
      '/collection?status=owned&q=kaiju+prime&sort=price_desc',
    )
  })

  it('leaves out a blank search and the default order, so a plain gallery stays plain', () => {
    expect(galleryHref({ q: '   ', sort: 'added_desc' })).toBe('/collection')
  })

  it('counts a search as narrowing, and an order as not', () => {
    expect(isFiltered({ q: 'kaiju' })).toBe(true)
    expect(isFiltered({ q: '  ' })).toBe(false)
    expect(isFiltered({ sort: 'name_asc' })).toBe(false)
  })
})

describe('SearchAndSort', () => {
  it('searches on Enter, keeping the other filters and dropping the page reference (FR-011, FR-012)', async () => {
    current = new URLSearchParams('status=owned&series=Kaiju+Wars&cursor=abc&deleted=x')
    const user = userEvent.setup()
    render(<SearchAndSort q={null} sort={null} />)

    await user.type(screen.getByRole('searchbox', { name: 'Search your vault' }), '  prime 1 {Enter}')

    expect(push).toHaveBeenCalledWith('/collection?status=owned&series=Kaiju+Wars&q=prime+1')
  })

  it('applies an order on change, and drops the default from the address', async () => {
    current = new URLSearchParams('q=kaiju&sort=price_desc')
    const user = userEvent.setup()
    render(<SearchAndSort q="kaiju" sort="price_desc" />)

    const select = screen.getByRole('combobox', { name: 'Sort by' })
    expect(select).toHaveValue('price_desc')
    await user.selectOptions(select, 'name_asc')
    expect(push).toHaveBeenLastCalledWith('/collection?q=kaiju&sort=name_asc')

    await user.selectOptions(select, 'added_desc')
    expect(push).toHaveBeenLastCalledWith('/collection?q=kaiju')
  })

  it('offers all ten orders by name', () => {
    render(<SearchAndSort q={null} sort={null} />)
    expect(screen.getAllByRole('option')).toHaveLength(10)
    expect(screen.getByRole('option', { name: 'Price, highest first' })).toBeInTheDocument()
  })

  it('clears an active search (US1/AC4)', async () => {
    current = new URLSearchParams('q=kaiju&status=sold')
    const user = userEvent.setup()
    render(<SearchAndSort q="kaiju" sort={null} />)

    await user.click(screen.getByRole('button', { name: 'Clear' }))
    expect(push).toHaveBeenCalledWith('/collection?status=sold')
  })
})

describe('NoResults for a search (FR-005)', () => {
  it('names the search that found nothing, and is not the empty-vault message', () => {
    render(<NoResults filter={{ q: 'zeppelin' }} total={312} />)
    const state = screen.getByTestId('no-results')
    expect(state).toHaveTextContent('Nothing matching “zeppelin”')
    expect(state).toHaveTextContent('Your vault holds 312 collectibles')
  })
})
