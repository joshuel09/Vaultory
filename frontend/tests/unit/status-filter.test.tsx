import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { StatusFilter } from '@/components/collection/StatusFilter'

const push = vi.fn()
vi.mock('next/navigation', () => ({
  useRouter: () => ({ push }),
  usePathname: () => '/collection',
  useSearchParams: () => new URLSearchParams(''),
}))

beforeEach(() => push.mockClear())

describe('StatusFilter', () => {
  // FR-037, FR-038: all four statuses, plus a way back to everything.
  it('offers every status and an All option', () => {
    render(<StatusFilter active={null} />)
    for (const label of ['All', 'Owned', 'Preordered', 'Wishlist', 'Sold']) {
      expect(screen.getByText(label)).toBeInTheDocument()
    }
  })

  // FR-039: the active filter is indicated, and not by colour alone — it is a checked radio, which
  // assistive technology reports.
  it('marks the active filter in a way assistive technology can report', () => {
    render(<StatusFilter active="preordered" />)
    const radios = screen.getAllByRole('radio')
    const checked = radios.filter((r) => (r as HTMLInputElement).checked)
    expect(checked).toHaveLength(1)
    expect(screen.getByText(/\(active filter\)/i)).toBeInTheDocument()
  })

  it('narrows to a status in one action', async () => {
    const user = userEvent.setup()
    render(<StatusFilter active={null} />)
    await user.click(screen.getByText('Sold'))
    expect(push).toHaveBeenCalledWith('/collection?status=sold')
  })

  // FR-038: returning to everything is also one action.
  it('returns to all statuses in one action', async () => {
    const user = userEvent.setup()
    render(<StatusFilter active="sold" />)
    await user.click(screen.getByText('All'))
    expect(push).toHaveBeenCalledWith('/collection')
  })

  // FR-045: operable by keyboard. A radiogroup gives this its proper semantics.
  it('is a labelled radio group', () => {
    render(<StatusFilter active={null} />)
    expect(screen.getByRole('radiogroup', { name: /filter by status/i })).toBeInTheDocument()
    expect(screen.getAllByRole('radio')).toHaveLength(5)
  })
})
