/**
 * The text of the messages Vaultory sends.
 *
 * Plain and transactional. Each says what it is for, who it is for, how long the link lasts, and
 * what to do if they did not ask for it (FR-024) — that last line matters most when the message
 * is unexpected, because an unexpected reset mail is the first sign somebody is trying an address.
 */

export function verificationMessage(email: string, url: string) {
  return {
    to: email,
    subject: 'Verify your email address',
    text: [
      `Hello,`,
      ``,
      `You created a Vaultory account with this address (${email}). Confirming it means you can`,
      `recover your vault later if you forget your password.`,
      ``,
      url,
      ``,
      `This link works for 24 hours. It confirms your address and nothing else — it will not sign`,
      `you in.`,
      ``,
      `If you did not create a Vaultory account, you can ignore this message. Nothing has been`,
      `created in your name that you need to undo.`,
      ``,
      `— Vaultory`,
    ].join('\n'),
  }
}

export function resetMessage(email: string, url: string) {
  return {
    to: email,
    subject: 'Reset your Vaultory password',
    text: [
      `Hello,`,
      ``,
      `Somebody asked to reset the password for the Vaultory account at ${email}.`,
      ``,
      url,
      ``,
      `This link works for one hour and can be used once. Choosing a new password will sign out`,
      `every other device currently signed in to your vault.`,
      ``,
      `If you did not ask for this, you can ignore this message — your password has not changed.`,
      `Nobody can use this link without access to this inbox.`,
      ``,
      `— Vaultory`,
    ].join('\n'),
  }
}
