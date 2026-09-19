import { betterAuth } from 'better-auth'
import { Pool } from 'pg'

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
export const auth = betterAuth({
  database: new Pool({ connectionString: process.env.BETTER_AUTH_DATABASE_URL }),
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
  emailAndPassword: {
    enabled: true,
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
      '/sign-in/email': { window: 15 * 60, max: 10 },
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
