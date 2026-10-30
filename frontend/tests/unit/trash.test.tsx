import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { UndoDeleteNotice } from '@/components/collection/UndoDeleteNotice'
import { TrashList } from '@/components/collection/TrashList'
import type { TrashedCollectible } from '@/lib/api/types'
import { aCollectible, anImage } from './fixtures'

// Feature 010: the Undo notice and the Trash.

const replace = vi.fn()
const refresh = vi.fn()
vi.mock('next/navigation', () => ({
  useRouter: () => ({ replace, refresh, push: vi.fn() }),
}))

function jsonResponse(status: number, body: unknown): Response {
  return { ok: status >= 200 && status < 300, status, json: async () => body } as Response
}

let fetchMock: ReturnType<typeof vi.fn>

beforeEach(() => {
  replace.mockClear()
  refresh.mockClear()
  fetchMock = vi.fn()
  vi.stubGlobal('fetch', fetchMock)
  // jsdom has <dialog> but not showModal/close; see delete-dialog.test.tsx for why this is a stub
  // and where the real browser behaviour is checked instead.
  if (!HTMLDialogElement.prototype.showModal) {
    HTMLDialogElement.prototype.showModal = function showModal(this: HTMLDialogElement) {
      this.open = true
    }
    HTMLDialogElement.prototype.close = function close(this: HTMLDialogElement) {
      this.open = false
      this.dispatchEvent(new Event('close'))
    }
  }
  if (!HTMLFormElement.prototype.requestSubmit) {
    HTMLFormElement.prototype.requestSubmit = function requestSubmit(this: HTMLFormElement) {
      this.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    }
  }
})

afterEach(() => vi.unstubAllGlobals())

const ID = '6f9619ff-8b86-d011-b42d-00cf4fc964ff'

function trashed(overrides: Partial<TrashedCollectible> = {}): TrashedCollectible {
  return {
    collectible: aCollectible({ id: ID, name: 'Kaiju Sentinel', image: anImage }),
    deletedAt: '2026-10-25T09:00:00Z',
    purgeAt: '2026-11-24T09:00:00Z',
    ...overrides,
  }
}

describe('UndoDeleteNotice', () => {
  it('says where it went and restores it on Undo, leaving the marker behind (FR-005, FR-007)', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, aCollectible({ id: ID })))
    const user = userEvent.setup()
    render(<UndoDeleteNotice deletedId={ID} galleryHref="/collection?status=owned" />)

    expect(screen.getByTestId('undo-delete-notice')).toHaveTextContent(/Trash/)
    expect(screen.getByRole('link', { name: 'Trash' })).toHaveAttribute('href', '/collection/trash')

    await user.click(screen.getByRole('button', { name: 'Undo' }))

    await waitFor(() => expect(replace).toHaveBeenCalledWith('/collection?status=owned', { scroll: false }))
    expect(refresh).toHaveBeenCalled()
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe(`/api/trash/${ID}/restore`)
    expect(init.method).toBe('POST')
  })

  it('says plainly when it could not come back, and stays (FR-008)', async () => {
    fetchMock.mockResolvedValue(jsonResponse(404, { error: { code: 'not_found', message: 'Not found.' } }))
    const user = userEvent.setup()
    render(<UndoDeleteNotice deletedId={ID} galleryHref="/collection" />)

    await user.click(screen.getByRole('button', { name: 'Undo' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(/could not be brought back/i)
    expect(replace).not.toHaveBeenCalled()
  })

  it('dismissing clears the marker without restoring', async () => {
    const user = userEvent.setup()
    render(<UndoDeleteNotice deletedId={ID} galleryHref="/collection" />)

    await user.click(screen.getByRole('button', { name: 'Dismiss' }))

    expect(replace).toHaveBeenCalledWith('/collection', { scroll: false })
    expect(fetchMock).not.toHaveBeenCalled()
  })
})

describe('TrashList', () => {
  it('shows each item with its photograph and the server-given date it goes for good (FR-006)', () => {
    render(<TrashList initialItems={[trashed()]} initialCursor={null} />)

    const item = screen.getByTestId('trashed-collectible')
    expect(within(item).getByText('Kaiju Sentinel')).toBeInTheDocument()
    expect(within(item).getByRole('img')).toHaveAttribute('src', anImage.renditionUrl)
    expect(within(item).getByText(/Removed for good on/)).toBeInTheDocument()
    expect(item.querySelector('time')).toHaveAttribute('dateTime', '2026-11-24T09:00:00Z')
  })

  it('explains the 30 days when the Trash is empty (US2/AC4)', () => {
    render(<TrashList initialItems={[]} initialCursor={null} />)
    expect(screen.getByTestId('empty-trash')).toHaveTextContent(/30 days/)
  })

  it('restores an item and removes it from the list (FR-007)', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, aCollectible({ id: ID })))
    const user = userEvent.setup()
    render(<TrashList initialItems={[trashed()]} initialCursor={null} />)

    await user.click(screen.getByRole('button', { name: 'Restore Kaiju Sentinel' }))

    expect(await screen.findByTestId('trash-message')).toHaveTextContent(/back in your vault/)
    expect(screen.queryByTestId('trashed-collectible')).not.toBeInTheDocument()
  })

  it('asks before deleting permanently, defaults to keeping it, and deletes only on confirm (FR-011)', async () => {
    fetchMock.mockResolvedValue({ ok: true, status: 204, json: async () => ({}) } as Response)
    const user = userEvent.setup()
    render(<TrashList initialItems={[trashed()]} initialCursor={null} />)

    await user.click(screen.getByRole('button', { name: 'Delete Kaiju Sentinel permanently' }))
    const dialog = screen.getByTestId('delete-permanently-dialog')
    expect(dialog).toHaveTextContent('Kaiju Sentinel')
    expect(dialog).toHaveTextContent(/cannot be undone/i)
    expect(screen.getByTestId('cancel-delete-permanently')).toHaveFocus()
    expect(fetchMock).not.toHaveBeenCalled()

    await user.click(screen.getByTestId('confirm-delete-permanently'))

    await waitFor(() => expect(screen.queryByTestId('trashed-collectible')).not.toBeInTheDocument())
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe(`/api/trash/${ID}`)
    expect(init.method).toBe('DELETE')
  })
})
