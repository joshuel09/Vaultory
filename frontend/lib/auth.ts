import { betterAuth } from 'better-auth'
import { Pool } from 'pg'
import { send } from '@/lib/mail'
import { verificationMessage, resetMessage } from '@/lib/messages'
import { createHash } from 'node:crypto'

/**
 * Better Auth owns registration, sign-in, password hashing, and session issuance (issue #15).
 *
 * It does NOT decide who may see a collection. The Go service verifies every session itself and
 * remains the only thing that authorizes access to a vault — see
 * specs/004-collector-authentication/contracts/README.md.
 *
 * The connection below uses a restricted role with no access to any collection table, so a bug
 * here cannot read a collection even though this process holds a database handle.
 */
const pool = new Pool({ connectionString: process.env.BETTER_AUTH_DATABASE_URL })

/**
 * How Better Auth stores a token when `storeIdentifier: 'hashed'` is set: SHA-256 of the
 * identifier, base64url, unpadded. Reproduced here so the row belonging to a *particular* token can
 * be told apart from the rest without ever storing or comparing the token itself.
 */
function storedIdentifier(identifier: string): string {
  return createHash('sha256').update(identifier, 'utf8').digest('base64url')
}

/**
 * Invalidate every outstanding reset link for an account except the one just issued (FR-012).
 *
 * Better Auth does not do this: `forget-password` inserts a row and deletes nothing, so with the
 * three-per-hour limit three working reset links could exist at once, each for an hour. A stale
 * verification link is harmless — it sets a boolean that is already true — but a stale reset link
 * is a way into a vault, which is why this one is implemented rather than narrowed away.
 *
 * `verification.value` holds the account id, so prior rows are findable even though identifiers
 * are hashed.
 */
async function invalidatePreviousResets(userId: string, currentToken: string): Promise<void> {
  await pool.query(
    `DELETE FROM verification
      WHERE value = $1
        AND identifier <> $2
        AND identifier NOT LIKE 'email-verification%'`,
    [userId, storedIdentifier(`reset-password:${currentToken}`)],
  )
}

export const auth = betterAuth({
  database: pool,
  secret: process.env.BETTER_AUTH_SECRET,
  baseURL: process.env.BETTER_AUTH_URL,
  /*
   * Every origin this application is legitimately reached on.
   *
   * Better Auth compares a request's Origin header against these and answers 403 INVALID_ORIGIN
   * otherwise — a CSRF defence, and a correct one. It bites whenever the origin in the browser's
   * address bar differs from baseURL: the browser suite reaches the app as http://frontend:3000
   * inside the Compose network, and in production this must list the origin collectors actually
   * use, or every sign-in fails with a message that says nothing about configuration.
   *
   * curl does not send an Origin header, which is why this is invisible until a real browser
   * tries it.
   */
  trustedOrigins: (process.env.BETTER_AUTH_TRUSTED_ORIGINS ?? '')
    .split(',')
    .map((origin) => origin.trim())
    .filter(Boolean),
  /*
   * Reset and verification tokens are stored hashed, not as issued (FR-015).
   *
   * Better Auth's default is 'plain', which writes the token itself into
   * verification.identifier — so a copy of that table would be a set of working reset links to
   * every account with an outstanding reset, and nothing about the running system would look
   * wrong. Hashed, the stored value can confirm a token presented to it and cannot produce one.
   *
   * No salt, deliberately: the token is many bytes of randomness, so there is nothing to
   * precompute against, and adding one would imply it needed stretching (research Decision 1).
   */
  verification: {
    storeIdentifier: 'hashed',
  },
  emailVerification: {
    sendOnSignUp: true,
    // 24 hours. The default is one, and FR-003 allows a day — which is what somebody who
    // registers at night and reads their mail the next morning needs (research Decision 5).
    expiresIn: 60 * 60 * 24,
    /*
     * autoSignInAfterVerification is deliberately NOT set (FR-002a).
     *
     * It is opt-in, and turning it on is exactly the convenience a later contributor would add.
     * A verification link lives 24 hours; treating it as a way in would make a forwarded or
     * archived message access to somebody's vault for a day. Proving you can read an inbox is not
     * proving you know a password. tests/unit/auth-config.test.ts guards this absence.
     */
    sendVerificationEmail: async ({ user, url }) => {
      // Point the callback at Vaultory's own confirmation page. Better Auth defaults it to "/",
      // which verifies the address correctly and then drops the collector on the landing page with
      // no indication anything happened — found by following a real link rather than assuming.
      const target = new URL(url)
      target.searchParams.set('callbackURL', '/verify-email')
      await send(verificationMessage(user.email, target.toString()))
    },
  },
  emailAndPassword: {
    enabled: true,
    /*
     * Ending every other session is the difference between a reset and a rename (FR-018).
     *
     * This is off by default. Left alone, a reset changes the password and leaves every existing
     * session working — so somebody resetting because they fear another person has their password
     * would not evict them. Nothing is needed in the Go service for it: revocation deletes the
     * session rows, and the resolver already requires a live row on every request.
     */
    revokeSessionsOnPasswordReset: true,
    // One hour, which is already the default (FR-010). Deliberately shorter than a verification
    // link, because a reset link is worth far more to whoever holds it.
    resetPasswordTokenExpiresIn: 60 * 60,
    sendResetPassword: async ({ user, url, token }) => {
      await invalidatePreviousResets(user.id, token)
      /*
       * The callback carries the address as well as the token, so the reset page can sign the
       * collector in once they have chosen a new password (FR-014). resetPassword returns only
       * `{status:true}` — no user — so there is otherwise nothing to sign in *as*.
       *
       * The address is not a credential, and this link already carries the token, which is. It
       * goes only to the inbox that owns the address.
       */
      const target = new URL(url)
      target.searchParams.set('callbackURL', `/reset-password?email=${encodeURIComponent(user.email)}`)
      await send(resetMessage(user.email, target.toString()))
    },
    /*
     * Completing a reset verifies the address if it was not already (FR-014a). Following a link
     * sent there proves control of the inbox, which is exactly what verification tests — asking
     * them to prove it again would be asking for something they have just demonstrated.
     */
    onPasswordReset: async ({ user }) => {
      await pool.query(
        'UPDATE "user" SET "emailVerified" = true, "updatedAt" = now() WHERE id = $1 AND "emailVerified" = false',
        [user.id],
      )
    },
    // FR-004. Length only: a length rule is honest about what it buys, where a composition rule
    // mostly teaches people to end passwords with "1!".
    minPasswordLength: 12,
  },
  rateLimit: {
    // FR-027: 10 failed sign-ins per account per 15 minutes, then 15 minutes of refusal.
    //
    // Stored in the database rather than in memory on purpose. In-memory counters reset on every
    // restart and are not shared between instances, so a limit that exists to slow guessing would
    // quietly evaporate on each deploy.
    enabled: true,
    storage: 'database',
    // The global ceiling is abuse protection, not the requirement, and it counts every auth path
    // per IP — including /get-session, which fires on each navigation. Set too low it throttles a
    // household or office behind one address, and it throttled the browser suite at 100.
    window: 15 * 60,
    max: 2000,
    customRules: {
      // FR-027, and the rule that actually matters: ten failed sign-ins per account per fifteen
      // minutes, then fifteen minutes of refusal that lifts by itself. Never a permanent lock —
      // password reset is out of scope, so a locked-out collector would have no way back in.
      /*
       * FR-027: ten failed sign-ins per fifteen minutes.
       *
       * Better Auth keys rate limits by IP, not by account, so this is "ten per address" rather
       * than the "ten per account" FR-027 describes. That is a real gap in feature 004's
       * implementation, not a decision made here — it is recorded in the spec rather than left for
       * somebody to discover. In practice it is stricter for shared networks and looser for an
       * attacker with many addresses.
       *
       * Configurable because the browser suite runs from one container: every account it creates
       * shares an address, and a limit meant for one person throttles the whole run.
       */
      '/sign-in/email': {
        window: 15 * 60,
        max: Number(process.env.BETTER_AUTH_SIGNIN_MAX ?? 10),
      },
      /*
       * Registration has to be stated explicitly. Better Auth applies its own stricter default to
       * this path, which the global ceiling above does not override — it refused the fourth
       * registration from one address, and the browser suite, which creates an account per test
       * from a single container, could not get past it.
       *
       * Twenty per fifteen minutes per address: comfortable for a household or office behind one
       * NAT address, tight enough to make bulk account creation tedious. Raised in development so
       * the suite can run.
       */
      '/sign-up/email': {
        window: 15 * 60,
        max: Number(process.env.BETTER_AUTH_SIGNUP_MAX ?? 20),
      },
      /*
       * FR-020: three recovery messages per hour, verification and reset counted separately.
       *
       * Separately on purpose — somebody who has exhausted verification resends must still be
       * able to request a reset, which is the moment they are most likely to need one.
       *
       * Stated explicitly rather than left to the global ceiling, for the reason feature 004
       * learned the hard way: Better Auth applies its own stricter defaults to some paths, and the
       * global setting does not override them. Here it silently throttled the browser suite until
       * these rules were added.
       */
      '/request-password-reset': {
        window: 60 * 60,
        max: Number(process.env.BETTER_AUTH_RECOVERY_MAX ?? 3),
      },
      '/send-verification-email': {
        window: 60 * 60,
        max: Number(process.env.BETTER_AUTH_RECOVERY_MAX ?? 3),
      },
      // Following a link is not requesting a message. Refusing these would lock somebody out of a
      // link they legitimately hold, so they are bounded generously rather than tightly.
      '/reset-password': { window: 60 * 60, max: 60 },
      '/verify-email': { window: 60 * 60, max: 60 },
    },
  },
  session: {
    // FR-010: 30 days from last use, extended as the collector keeps using Vaultory. The 90-day
    // ceiling is enforced by Go at verification time, because Better Auth has no notion of one
    // (research Decision 6).
    expiresIn: 60 * 60 * 24 * 30,
    updateAge: 60 * 60 * 24,
  },
})
