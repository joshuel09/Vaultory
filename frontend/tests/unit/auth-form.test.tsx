import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AuthForm } from '@/components/auth/AuthForm'

const push = vi.fn()
const refresh = vi.fn()
vi.mock('next/navigation', () => ({ useRouter: () => ({ push, refresh }) }))

beforeEach(() => {
  push.mockClear()
  refresh.mockClear()
})

function setup(onSubmit = vi.fn().mockResolvedValue({ error: null })) {
  render(<AuthForm mode="register" submitLabel="Create my vault" next="/collection" onSubmit={onSubmit} />)
  return { onSubmit, user: userEvent.setup() }
}

const emailField = () => screen.getByLabelText(/^email/i)
const passwordField = () => screen.getByLabelText(/^password/i)
const submit = () => screen.getByRole('button', { name: /create my vault/i })

describe('the credential form', () => {
  // FR-026: every problem at once, not one per attempt.
  it('reports both empty fields together', async () => {
    const { onSubmit, user } = setup()
    await user.click(submit())

    expect(await screen.findByText(/an email address is required/i)).toBeInTheDocument()
    expect(screen.getByText(/a password is required/i)).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  // FR-004, and the message says what is required rather than only that it is wrong.
  it('refuses a password under 12 characters and says so', async () => {
    const { onSubmit, user } = setup()
    await user.type(emailField(), 'collector@example.com')
    await user.type(passwordField(), 'short')
    await user.click(submit())

    expect(await screen.findByText(/at least 12 characters/i)).toBeInTheDocument()
    expect(onSubmit).not.toHaveBeenCalled()
  })

  // FR-003: uniqueness and sign-in are case-insensitive, so the address is normalised before it
  // ever leaves the browser.
  it('lower-cases and trims the email before submitting', async () => {
    const { onSubmit, user } = setup()
    await user.type(emailField(), '  Collector@Example.COM  ')
    await user.type(passwordField(), 'a-long-enough-password')
    await user.click(submit())

    expect(onSubmit).toHaveBeenCalledWith({
      email: 'collector@example.com',
      password: 'a-long-enough-password',
    })
  })

  // FR-026 again, and the part that matters most: a failed submit must not cost someone what they
  // typed.
  it('keeps what was typed when the server refuses', async () => {
    const onSubmit = vi.fn().mockResolvedValue({ error: { message: 'That email is already registered.' } })
    const { user } = setup(onSubmit)
    await user.type(emailField(), 'taken@example.com')
    await user.type(passwordField(), 'a-long-enough-password')
    await user.click(submit())

    expect(await screen.findByTestId('form-error')).toHaveTextContent(/already registered/i)
    expect(emailField()).toHaveValue('taken@example.com')
    expect(passwordField()).toHaveValue('a-long-enough-password')
    expect(push).not.toHaveBeenCalled()
  })

  it('announces failures to assistive technology', async () => {
    const onSubmit = vi.fn().mockResolvedValue({ error: { message: 'Nope.' } })
    const { user } = setup(onSubmit)
    await user.type(emailField(), 'collector@example.com')
    await user.type(passwordField(), 'a-long-enough-password')
    await user.click(submit())

    expect(await screen.findByRole('alert')).toBeInTheDocument()
  })

  it('goes where it was told once accepted', async () => {
    const { user } = setup()
    await user.type(emailField(), 'collector@example.com')
    await user.type(passwordField(), 'a-long-enough-password')
    await user.click(submit())

    await vi.waitFor(() => expect(push).toHaveBeenCalledWith('/collection'))
  })
})
