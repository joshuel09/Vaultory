import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { DeleteCollectibleDialog } from '@/components/collection/DeleteCollectibleDialog'

let fetchMock: ReturnType<typeof vi.fn>
const onDeleted = vi.fn()

beforeEach(() => {
  onDeleted.mockClear()
  fetchMock = vi.fn()
  vi.stubGlobal('fetch', fetchMock)

  /*
   * jsdom knows the <dialog> element but implements neither showModal/close nor form submission,
   * so both are supplied here.
   *
   * What that means for the two tests below that press Enter or Cancel: in a real browser those
   * submit a form with method="dialog", which closes it without deleting. jsdom would otherwise
   * throw "not implemented", and the tests would pass because nothing happened at all rather than
   * because the right thing happened. The stub closes the dialog, so the assertion is real — but
   * the *browser's* behaviour of method="dialog" is verified end to end in
   * tests/e2e/delete-collectible.spec.ts, not here. A unit test cannot prove a platform behaviour
   * it has had to stub.
   */
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

function open() {
  return render(
    <DeleteCollectibleDialog collectibleId="abc-123" name="Kaiju Sentinel" onDeleted={onDeleted} />,
  )
}

describe('DeleteCollectibleDialog', () => {
  it('names the collectible and says the deletion cannot be undone (FR-023)', async () => {
    const user = userEvent.setup()
    open()
    await user.click(screen.getByTestId('delete-collectible'))

    const dialog = screen.getByTestId('delete-dialog')
    expect(dialog).toHaveTextContent('Kaiju Sentinel')
    expect(dialog).toHaveTextContent(/cannot be undone/i)
  })

  it('focuses the way out, not the way through (FR-023, FR-037)', async () => {
    const user = userEvent.setup()
    open()
    await user.click(screen.getByTestId('delete-collectible'))

    // The destructive button must never be what a stray keypress lands on.
    expect(screen.getByTestId('cancel-delete')).toHaveFocus()
    expect(screen.getByTestId('confirm-delete')).not.toHaveFocus()
  })

  it('deletes nothing when Enter is pressed on an untouched dialog (FR-023)', async () => {
    const user = userEvent.setup()
    open()
    await user.click(screen.getByTestId('delete-collectible'))
    await user.keyboard('{Enter}')

    // The assertion that matters: no request left the building. Whether the dialog also closed is
    // the browser's method="dialog" behaviour, which only the end-to-end suite can show.
    expect(fetchMock).not.toHaveBeenCalled()
    expect(onDeleted).not.toHaveBeenCalled()
  })

  it('deletes nothing when the collector cancels', async () => {
    const user = userEvent.setup()
    open()
    await user.click(screen.getByTestId('delete-collectible'))
    await user.click(screen.getByTestId('cancel-delete'))

    expect(fetchMock).not.toHaveBeenCalled()
    expect(onDeleted).not.toHaveBeenCalled()
  })

  it('deletes only when the collector confirms', async () => {
    fetchMock.mockResolvedValue({ ok: true, status: 204 } as Response)
    const user = userEvent.setup()
    open()
    await user.click(screen.getByTestId('delete-collectible'))
    await user.click(screen.getByTestId('confirm-delete'))

    await waitFor(() => expect(onDeleted).toHaveBeenCalled())
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toContain('/api/collectibles/abc-123')
    expect(init.method).toBe('DELETE')
  })

  it('issues exactly one request when confirm is double-clicked (FR-036)', async () => {
    let release: (r: Response) => void = () => {}
    fetchMock.mockReturnValue(new Promise<Response>((resolve) => { release = resolve }))

    const user = userEvent.setup()
    open()
    await user.click(screen.getByTestId('delete-collectible'))
    const confirm = screen.getByTestId('confirm-delete')
    await user.click(confirm)
    await user.click(confirm)
    await user.click(confirm)

    expect(fetchMock).toHaveBeenCalledTimes(1)
    release({ ok: true, status: 204 } as Response)
    await waitFor(() => expect(onDeleted).toHaveBeenCalledTimes(1))
  })

  it('says nothing changed when the deletion fails, and does not navigate', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 500,
      json: async () => ({ error: { code: 'internal_error', message: 'Something went wrong.' } }),
    } as Response)
    const user = userEvent.setup()
    open()
    await user.click(screen.getByTestId('delete-collectible'))
    await user.click(screen.getByTestId('confirm-delete'))

    expect(await screen.findByRole('alert')).toHaveTextContent(/nothing has changed/i)
    expect(onDeleted).not.toHaveBeenCalled()
  })
})
