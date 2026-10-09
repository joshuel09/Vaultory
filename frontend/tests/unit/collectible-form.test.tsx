import { act, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { EditCollectibleForm } from '@/components/collection/EditCollectibleForm'
import { aCollectible, anImage } from './fixtures'

const push = vi.fn()
vi.mock('next/navigation', () => ({
  useRouter: () => ({ push, refresh: vi.fn() }),
}))

function jsonResponse(status: number, body: unknown): Response {
  return { ok: status >= 200 && status < 300, status, json: async () => body } as Response
}

let fetchMock: ReturnType<typeof vi.fn>

beforeEach(() => {
  push.mockClear()
  fetchMock = vi.fn()
  vi.stubGlobal('fetch', fetchMock)
})

afterEach(() => vi.unstubAllGlobals())

/** The body of a PUT the form sent. */
function sentBody(call = 0): Record<string, unknown> {
  const init = fetchMock.mock.calls[call]?.[1] as RequestInit | undefined
  if (!init?.body) throw new Error(`no request body for call ${call}`)
  return JSON.parse(init.body as string)
}

const stored = aCollectible({
  version: 3,
  name: 'Kaiju Sentinel',
  collectionStatus: 'preordered',
  series: 'Kaiju Wars',
  purchasePrice: '1250.00',
})

describe('EditCollectibleForm', () => {
  it('opens with every stored value already filled in (FR-002)', () => {
    render(<EditCollectibleForm collectible={stored} returnTo="/collection" />)

    expect(screen.getByLabelText(/^name/i)).toHaveValue('Kaiju Sentinel')
    expect(screen.getByLabelText(/series/i)).toHaveValue('Kaiju Wars')
    expect(screen.getByLabelText(/purchase price/i)).toHaveValue('1250.00')
    expect(screen.getByLabelText(/collection status/i)).toHaveValue('preordered')
  })

  it('leaves attributes the collector never supplied empty rather than defaulted (FR-003)', () => {
    render(<EditCollectibleForm collectible={stored} returnTo="/collection" />)

    // stored has no character, manufacturer, scale or notes.
    expect(screen.getByLabelText(/character/i)).toHaveValue('')
    expect(screen.getByLabelText(/manufacturer/i)).toHaveValue('')
    expect(screen.getByLabelText(/scale/i)).toHaveValue('')
    expect(screen.getByLabelText(/notes/i)).toHaveValue('')
  })

  it('sends the version it was opened at (FR-027a)', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { ...stored, version: 4 }))
    const user = userEvent.setup()
    render(<EditCollectibleForm collectible={stored} returnTo="/collection" />)

    await user.click(screen.getByRole('button', { name: /save changes/i }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())

    expect(sentBody().expectedVersion).toBe(3)
  })

  it('sends the current image back when the photograph is untouched (FR-018)', async () => {
    const withPhoto = aCollectible({ version: 2, image: anImage })
    fetchMock.mockResolvedValue(jsonResponse(200, withPhoto))
    const user = userEvent.setup()
    render(<EditCollectibleForm collectible={withPhoto} returnTo="/collection" />)

    await user.clear(screen.getByLabelText(/^name/i))
    await user.type(screen.getByLabelText(/^name/i), 'Renamed')
    await user.click(screen.getByRole('button', { name: /save changes/i }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())

    // The sharpest edge in the contract: a full replacement means omitting this removes the
    // photograph. An edit that only fixes a name must not destroy an image.
    expect(sentBody().imageId).toBe(anImage.id)
  })

  it('sends imageId null when the collector removes the photograph (FR-017)', async () => {
    const withPhoto = aCollectible({ version: 2, image: anImage })
    fetchMock.mockResolvedValue(jsonResponse(200, { ...withPhoto, image: null }))
    const user = userEvent.setup()
    render(<EditCollectibleForm collectible={withPhoto} returnTo="/collection" />)

    // The picker opens holding the stored photograph, and its Remove clears the reference rather
    // than only the preview — otherwise the save would send the old id straight back.
    await user.click(screen.getByRole('button', { name: /remove/i }))
    await user.click(screen.getByRole('button', { name: /save changes/i }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())

    expect(sentBody().imageId).toBeNull()
  })

  it('submits a cleared optional as absent rather than as an empty string (FR-005)', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, stored))
    const user = userEvent.setup()
    render(<EditCollectibleForm collectible={stored} returnTo="/collection" />)

    await user.clear(screen.getByLabelText(/purchase price/i))
    await user.click(screen.getByRole('button', { name: /save changes/i }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalled())

    expect(sentBody().purchasePrice).toBeNull()
  })

  it('issues exactly one request when the save is double-clicked (FR-036)', async () => {
    // Held open so the second click lands while the first is still in flight — which is the whole
    // point. Two PUTs carrying expectedVersion 3 would mean the second comes back 409 and tells
    // the collector the collectible changed, about their own save.
    let release: (r: Response) => void = () => {}
    fetchMock.mockReturnValue(new Promise<Response>((resolve) => { release = resolve }))

    const user = userEvent.setup()
    render(<EditCollectibleForm collectible={stored} returnTo="/collection" />)
    const save = screen.getByRole('button', { name: /save changes/i })

    await user.click(save)
    await user.click(save)
    await user.click(save)

    expect(fetchMock).toHaveBeenCalledTimes(1)

    // Let the held request finish inside the test, so the state update it causes is not left
    // landing after teardown.
    await act(async () => {
      release(jsonResponse(200, { ...stored, version: 4 }))
    })
  })

  it('keeps what the collector typed when the save fails (FR-035)', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse(400, {
        error: {
          code: 'validation_failed',
          message: 'This collectible could not be saved.',
          fields: [{ field: 'purchasePrice', code: 'negative_amount', message: 'A purchase price cannot be negative.' }],
        },
      }),
    )
    const user = userEvent.setup()
    render(<EditCollectibleForm collectible={stored} returnTo="/collection" />)

    await user.clear(screen.getByLabelText(/purchase price/i))
    await user.type(screen.getByLabelText(/purchase price/i), '-5.00')
    await user.clear(screen.getByLabelText(/^name/i))
    await user.type(screen.getByLabelText(/^name/i), 'Still here')
    await user.click(screen.getByRole('button', { name: /save changes/i }))

    expect(await screen.findByText(/cannot be negative/i)).toBeInTheDocument()
    // Losing a filled-in form to a failed save is the thing this must never do.
    expect(screen.getByLabelText(/^name/i)).toHaveValue('Still here')
    expect(screen.getByLabelText(/purchase price/i)).toHaveValue('-5.00')
  })

  it('shows what the collectible now says when somebody changed it first (FR-027)', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse(409, {
        error: { code: 'version_conflict', message: 'This collectible changed since you opened it.' },
        current: aCollectible({ version: 9, name: 'Changed In Another Tab', series: 'Kaiju Wars' }),
      }),
    )
    const user = userEvent.setup()
    render(<EditCollectibleForm collectible={stored} returnTo="/collection" />)

    await user.clear(screen.getByLabelText(/^name/i))
    await user.type(screen.getByLabelText(/^name/i), 'My edit')
    await user.click(screen.getByRole('button', { name: /save changes/i }))

    const notice = await screen.findByTestId('version-conflict')
    expect(notice).toHaveTextContent(/changed since you opened it/i)
    // Not merely "it changed" — the collector has to see how, or they have nothing to decide on.
    expect(notice).toHaveTextContent('Changed In Another Tab')
    // And their own work is still in front of them.
    expect(screen.getByLabelText(/^name/i)).toHaveValue('My edit')
    expect(push).not.toHaveBeenCalled()
  })

  it('adopts the current values only when the collector asks (FR-027)', async () => {
    fetchMock.mockResolvedValue(
      jsonResponse(409, {
        error: { code: 'version_conflict', message: 'Changed.' },
        current: aCollectible({ version: 9, name: 'Changed In Another Tab' }),
      }),
    )
    const user = userEvent.setup()
    render(<EditCollectibleForm collectible={stored} returnTo="/collection" />)

    await user.clear(screen.getByLabelText(/^name/i))
    await user.type(screen.getByLabelText(/^name/i), 'My edit')
    await user.click(screen.getByRole('button', { name: /save changes/i }))
    await screen.findByTestId('version-conflict')

    await user.click(screen.getByRole('button', { name: /use these values/i }))

    expect(screen.getByLabelText(/^name/i)).toHaveValue('Changed In Another Tab')
    expect(screen.queryByTestId('version-conflict')).not.toBeInTheDocument()

    // And the next save presents the version those values came from, not the stale one.
    fetchMock.mockResolvedValue(jsonResponse(200, aCollectible({ version: 10 })))
    await user.click(screen.getByRole('button', { name: /save changes/i }))
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2))
    expect(sentBody(1).expectedVersion).toBe(9)
  })

  it('returns the collector to where they were browsing, with one navigation', async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { ...stored, version: 4 }))
    const user = userEvent.setup()
    render(<EditCollectibleForm collectible={stored} returnTo="/collection?status=sold" />)

    await user.click(screen.getByRole('button', { name: /save changes/i }))

    await waitFor(() => expect(push).toHaveBeenCalledWith('/collection?status=sold'))
    expect(push).toHaveBeenCalledTimes(1)
  })
})
