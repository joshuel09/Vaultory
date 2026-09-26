import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const sendMail = vi.fn()
vi.mock('nodemailer', () => ({
  default: { createTransport: vi.fn(() => ({ sendMail })) },
  createTransport: vi.fn(() => ({ sendMail })),
}))

import nodemailer from 'nodemailer'
import { resetTransportForTests, send } from '@/lib/mail'

beforeEach(() => {
  sendMail.mockReset().mockResolvedValue({ messageId: 'x' })
  vi.mocked(nodemailer.createTransport).mockClear()
  resetTransportForTests()
  vi.stubEnv('MAIL_HOST', 'mailpit')
  vi.stubEnv('MAIL_PORT', '1025')
  vi.stubEnv('MAIL_FROM', 'Vaultory <no-reply@vaultory.local>')
})
afterEach(() => vi.unstubAllEnvs())

describe('send', () => {
  it('passes the recipient, subject and body through', async () => {
    await send({ to: 'collector@example.test', subject: 'Verify your address', text: 'link here' })

    expect(sendMail).toHaveBeenCalledWith({
      from: 'Vaultory <no-reply@vaultory.local>',
      to: 'collector@example.test',
      subject: 'Verify your address',
      text: 'link here',
    })
  })

  /**
   * The failure this test exists for. A send that fails quietly produces a page that looks
   * correct, an email that never arrives, and a bug report a week later — so the error has to
   * reach the caller rather than be swallowed here.
   */
  it('surfaces a transport failure rather than swallowing it', async () => {
    sendMail.mockRejectedValue(new Error('connection refused'))

    await expect(
      send({ to: 'collector@example.test', subject: 'x', text: 'y' }),
    ).rejects.toThrow(/connection refused/)
  })

  it('refuses to send when no host is configured, and says why', async () => {
    vi.stubEnv('MAIL_HOST', '')
    resetTransportForTests()

    await expect(send({ to: 'a@b.test', subject: 'x', text: 'y' })).rejects.toThrow(/MAIL_HOST/)
  })

  it('omits credentials when none are configured, so local SMTP needs no account', async () => {
    await send({ to: 'a@b.test', subject: 'x', text: 'y' })
    const config = vi.mocked(nodemailer.createTransport).mock.calls[0]?.[0] as Record<string, unknown>
    expect(config).not.toHaveProperty('auth')
    expect(config.host).toBe('mailpit')
  })

  it('sends credentials when they are configured', async () => {
    vi.stubEnv('MAIL_USER', 'postmaster')
    vi.stubEnv('MAIL_PASSWORD', 'secret')
    resetTransportForTests()

    await send({ to: 'a@b.test', subject: 'x', text: 'y' })
    const config = vi.mocked(nodemailer.createTransport).mock.calls[0]?.[0] as Record<string, unknown>
    expect(config.auth).toEqual({ user: 'postmaster', pass: 'secret' })
  })
})
