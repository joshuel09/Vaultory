import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { AddCollectibleForm } from '@/components/collection/AddCollectibleForm'

const refresh = vi.fn()
vi.mock('next/navigation', () => ({
  useRouter: () => ({ refresh, push: vi.fn() }),
}))

function jsonResponse(status: number, body: unknown): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as Response
}

let fetchMock: ReturnType<typeof vi.fn>

beforeEach(() => {
  refresh.mockClear()
  fetchMock = vi.fn()
  vi.stubGlobal('fetch', fetchMock)
  // The form generates one submission key per collectible (FR-047).
  vi.stubGlobal('crypto', { ...globalThis.crypto, randomUUID: () => 'key-under-test' })
})

afterEach(() => vi.unstubAllGlobals())

/** The body of the nth recorded fetch, asserted to exist so strict indexing is satisfied. */
function sentBody(call = 0): Record<string, unknown> {
  const recorded = fetchMock.mock.calls[call]
  if (!recorded) throw new Error(`no fetch call at index ${call}`)
  const init = recorded[1] as RequestInit
  return JSON.parse(init.body as string) as Record<string, unknown>
}

async function fillAndSubmit(user: ReturnType<typeof userEvent.setup>, name = 'Kaiju Sentinel') {
  await user.type(screen.getByLabelText(/^name/i), name)
  await user.click(screen.getByRole('button', { name: /add to my vault/i }))
}

describe('AddCollectibleForm', () => {
  // FR-005: a name and a status are all that is required.
  it('submits with only a name and the default status', async () => {
    const user = userEvent.setup()
    fetchMock.mockResolvedValue(jsonResponse(201, { id: '1', name: 'Kaiju Sentinel' }))
    render(<AddCollectibleForm />)

    await fillAndSubmit(user)

    await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce())
    const body = sentBody()
    expect(body.name).toBe('Kaiju Sentinel')
    expect(body.collectionStatus).toBe('owned')
    // FR-007: blank optional values are omitted, not sent as empty strings.
    expect(body.character).toBeUndefined()
    expect(body.purchasePrice).toBeUndefined()
  })

  // FR-047: the submission key rides along, and is the same on a retry of the same collectible.
  it('sends a submission key', async () => {
    const user = userEvent.setup()
    fetchMock.mockResolvedValue(jsonResponse(201, { id: '1', name: 'X' }))
    render(<AddCollectibleForm />)

    await fillAndSubmit(user, 'X')
    await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce())
    expect(sentBody().submissionKey).toBe('key-under-test')
  })

  // FR-021: a visible confirmation.
  it('confirms the collectible was added', async () => {
    const user = userEvent.setup()
    fetchMock.mockResolvedValue(jsonResponse(201, { id: '1', name: 'Kaiju Sentinel' }))
    render(<AddCollectibleForm />)

    await fillAndSubmit(user)

    expect(await screen.findByTestId('add-success')).toBeInTheDocument()
    expect(screen.getByText(/was added to your vault/i)).toBeInTheDocument()

    // Deliberately no assertion that router.refresh() was called.
    //
    // It used to be, and it was asserting an implementation detail that turned out to be both
    // redundant and harmful: the refresh re-rendered this static form, while the collection page
    // is force-dynamic and re-fetches on navigation anyway. Its only real effect was a navigation
    // that could abort the next one. What matters is the confirmation above.
  })

  // FR-020: every field error the server reported, shown against its own field.
  it('renders every server field error at once', async () => {
    const user = userEvent.setup()
    fetchMock.mockResolvedValue(
      jsonResponse(400, {
        error: {
          code: 'validation_failed',
          message: 'This collectible could not be saved.',
          fields: [
            { field: 'name', code: 'required', message: 'A name is required.' },
            { field: 'purchasePrice', code: 'negative_amount', message: 'A purchase price cannot be negative.' },
            { field: 'purchaseDate', code: 'date_in_future', message: 'A purchase date cannot be in the future.' },
          ],
        },
      }),
    )
    render(<AddCollectibleForm />)

    await fillAndSubmit(user, 'Something')

    expect(await screen.findByText('A name is required.')).toBeInTheDocument()
    expect(screen.getByText('A purchase price cannot be negative.')).toBeInTheDocument()
    expect(screen.getByText('A purchase date cannot be in the future.')).toBeInTheDocument()
  })

  // FR-022: a failed save must not cost the collector what they typed.
  it('preserves every entered value when saving fails', async () => {
    const user = userEvent.setup()
    fetchMock.mockResolvedValue(
      jsonResponse(500, { error: { code: 'internal_error', message: 'Something went wrong.' } }),
    )
    render(<AddCollectibleForm />)

    await user.type(screen.getByLabelText(/^name/i), 'Kaiju Sentinel')
    await user.type(screen.getByLabelText(/character/i), 'Sentinel Prime')
    await user.type(screen.getByLabelText(/notes/i), 'Box has a dent.')
    await user.click(screen.getByRole('button', { name: /add to my vault/i }))

    expect(await screen.findByTestId('form-error')).toBeInTheDocument()
    // The decisive assertion: nothing was cleared.
    expect(screen.getByLabelText(/^name/i)).toHaveValue('Kaiju Sentinel')
    expect(screen.getByLabelText(/character/i)).toHaveValue('Sentinel Prime')
    expect(screen.getByLabelText(/notes/i)).toHaveValue('Box has a dent.')
  })

  // A field error clears as the collector addresses it, rather than lingering.
  it('clears a field error once the collector edits that field', async () => {
    const user = userEvent.setup()
    fetchMock.mockResolvedValue(
      jsonResponse(400, {
        error: {
          code: 'validation_failed',
          message: 'no',
          fields: [{ field: 'name', code: 'required', message: 'A name is required.' }],
        },
      }),
    )
    render(<AddCollectibleForm />)

    await user.click(screen.getByRole('button', { name: /add to my vault/i }))
    expect(await screen.findByText('A name is required.')).toBeInTheDocument()

    await user.type(screen.getByLabelText(/^name/i), 'Now it has one')
    expect(screen.queryByText('A name is required.')).not.toBeInTheDocument()
  })

  // FR-029: an expired session is explained, rather than reported as a generic failure.
  it('explains an expired session', async () => {
    const user = userEvent.setup()
    fetchMock.mockResolvedValue(
      jsonResponse(401, { error: { code: 'unauthenticated', message: 'Sign in to view your collection.' } }),
    )
    render(<AddCollectibleForm />)

    await fillAndSubmit(user)
    expect(await screen.findByText(/session has expired/i)).toBeInTheDocument()
  })

  // FR-045: every input is reachable by its label.
  it('labels every field', () => {
    render(<AddCollectibleForm />)
    for (const label of [
      /^name/i, /collection status/i, /character/i, /series or franchise/i,
      /manufacturer/i, /category/i, /scale/i, /edition or variant/i,
      /purchase price/i, /purchase date/i, /release date/i, /notes/i,
    ]) {
      expect(screen.getByLabelText(label)).toBeInTheDocument()
    }
  })

  it('offers exactly the four statuses', () => {
    render(<AddCollectibleForm />)
    const select = screen.getByLabelText(/collection status/i)
    expect(select.querySelectorAll('option')).toHaveLength(4)
  })
})
