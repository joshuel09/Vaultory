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
    window: 15 * 60,
    max: 100,
    customRules: {
      '/sign-in/email': { window: 15 * 60, max: 10 },
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
