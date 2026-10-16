import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ResetPasswordForm } from '@/components/auth/ResetPasswordForm'
import { ResetCompleteNotice } from '@/components/auth/ResetCompleteNotice'

const push = vi.fn()
const refresh = vi.fn()
vi.mock('next/navigation', () => ({ useRouter: () => ({ push, refresh }) }))

const resetPassword = vi.fn()
const signInEmail = vi.fn()
vi.mock('@/lib/auth-client', () => ({
  authClient: {
    resetPassword: (...args: unknown[]) => resetPassword(...args),
    signIn: { email: (...args: unknown[]) => signInEmail(...args) },
  },
}))

beforeEach(() => {
  push.mockClear()
  refresh.mockClear()
  resetPassword.mockReset().mockResolvedValue({ error: null })
  signInEmail.mockReset().mockResolvedValue({ error: null })
})

const NEW = 'a-brand-new-password-x'

async function submit(email: string | null = 'collector@example.com') {
  const user = userEvent.setup()
  render(<ResetPasswordForm token="tok" email={email} />)
  await user.type(screen.getByLabelText(/new password/i), NEW)
  await user.click(screen.getByRole('button', { name: /set my new password/i }))
  return user
}

describe('completing a password reset', () => {
  it('signs the collector in and takes them to their collection', async () => {
    await submit()

    expect(resetPassword).toHaveBeenCalledWith({ token: 'tok', newPassword: NEW })
    expect(signInEmail).toHaveBeenCalledWith({ email: 'collector@example.com', password: NEW })
    expect(push).toHaveBeenCalledWith('/collection')
  })

  /*
   * Issue #41, and the reason it exists: the result of the sign-in used to be discarded.
   *
   * Pushing to /collection without a session sends the collector through the middleware to
   * /sign-in?next=/collection, where they get a bare form and no word about whether their reset
   * worked — immediately after it did work. This asserts the push never goes there.
   */
  it('does not send them to their collection when the sign-in was refused', async () => {
    signInEmail.mockResolvedValue({ error: { status: 401, message: 'nope' } })
    await submit()

    expect(push).not.toHaveBeenCalledWith('/collection')
    expect(push).toHaveBeenCalledWith('/sign-in?reset=done')
  })

  // The likeliest cause in practice. Sign-in is limited per address, so somebody who mistyped
  // their old password a few times before resetting arrives with that budget already spent — and
  // telling them to sign in now would send them into a second refusal.
  it('distinguishes a throttled sign-in, because the advice differs', async () => {
    signInEmail.mockResolvedValue({ error: { status: 429, message: 'too many requests' } })
    await submit()

    expect(push).toHaveBeenCalledWith('/sign-in?reset=throttled')
  })

  it('still reports the reset when there is no address to sign in as', async () => {
    await submit(null)

    expect(signInEmail).not.toHaveBeenCalled()
    expect(push).toHaveBeenCalledWith('/sign-in?reset=done')
  })

  // A failed reset is a different thing entirely: the link was not spent, and the collector stays
  // where they are so they can try again.
  it('stays put and explains when the reset itself failed', async () => {
    resetPassword.mockResolvedValue({ error: { message: 'That link is no longer valid.' } })
    await submit()

    expect(signInEmail).not.toHaveBeenCalled()
    expect(push).not.toHaveBeenCalled()
    expect(await screen.findByTestId('form-error')).toHaveTextContent(/no longer valid/i)
  })

  // FR-013: too short is refused in the browser, before the one-use link is spent.
  it('refuses a short password without spending the link', async () => {
    const user = userEvent.setup()
    render(<ResetPasswordForm token="tok" email="collector@example.com" />)
    await user.type(screen.getByLabelText(/new password/i), 'short')
    await user.click(screen.getByRole('button', { name: /set my new password/i }))

    expect(resetPassword).not.toHaveBeenCalled()
    expect(screen.getByText(/at least 12 characters/i)).toBeInTheDocument()
  })
})

describe('what the collector is told on arriving at sign-in', () => {
  it('says the password was changed, so they do not reuse a spent link', () => {
    render(<ResetCompleteNotice throttled={false} />)

    expect(screen.getByTestId('reset-complete')).toHaveTextContent('Your password has been changed')
    expect(screen.getByTestId('reset-complete')).toHaveTextContent(/sign in with your new password/i)
  })

  it('tells a throttled collector to wait rather than to try now', () => {
    render(<ResetCompleteNotice throttled />)

    const notice = screen.getByTestId('reset-complete')
    expect(notice).toHaveTextContent('Your password has been changed')
    expect(notice).toHaveTextContent(/too many sign-in attempts/i)
    expect(notice).toHaveTextContent(/wait a few minutes/i)
  })

  // Announced, not merely drawn: somebody using a screen reader arrives at this page by redirect
  // and needs to be told why.
  it('is announced to assistive technology', () => {
    render(<ResetCompleteNotice throttled={false} />)
    expect(screen.getByTestId('reset-complete')).toHaveAttribute('role', 'status')
  })
})
