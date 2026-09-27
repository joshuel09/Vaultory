import nodemailer, { type Transporter } from 'nodemailer'

/**
 * Sending mail, behind one function.
 *
 * The transport is the only thing that differs between development and production: Mailpit locally,
 * a real SMTP service in production. No call site learns which one is in use, which is what makes
 * the development path worth trusting — it exercises the same code that ships.
 *
 * Nothing here logs the message body. Recovery mail contains links that are credentials, and a link
 * in a log is a credential in a log (FR-016).
 */
export interface Message {
  to: string
  subject: string
  text: string
}

let cached: Transporter | null = null

function transport(): Transporter {
  if (cached) return cached

  const host = process.env.MAIL_HOST
  if (!host) {
    throw new Error(
      'MAIL_HOST is not set. Recovery mail cannot be sent, and a send that silently goes ' +
        'nowhere is indistinguishable from one that worked until somebody checks their inbox.',
    )
  }

  const user = process.env.MAIL_USER
  const pass = process.env.MAIL_PASSWORD

  cached = nodemailer.createTransport({
    host,
    port: Number(process.env.MAIL_PORT ?? 1025),
    // Mailpit speaks plain SMTP on 1025; a real service will be on 587 with STARTTLS, which
    // nodemailer negotiates when secure is false and the server advertises it.
    secure: false,
    ...(user && pass ? { auth: { user, pass } } : {}),
  })
  return cached
}

export async function send({ to, subject, text }: Message): Promise<void> {
  await transport().sendMail({
    from: process.env.MAIL_FROM ?? 'Vaultory <no-reply@vaultory.local>',
    to,
    subject,
    text,
  })
}

/** Test seam. Resets the memoised transport so a test can change the environment. */
export function resetTransportForTests(): void {
  cached = null
}
