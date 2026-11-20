# Research: Lock Sign-In per Email Address

## 1. Where the rule lives

- **Decision**: three PostgreSQL functions — `sign_in_record_failure(key)`, `sign_in_is_locked(key)`, `sign_in_clear(key)` — over one table.
- **Rationale**: the count must be shared by every instance and survive restarts (FR-008), and the increment must not lose concurrent failures (SC-005). A single `INSERT … ON CONFLICT DO UPDATE` in a function is atomic under concurrency without application locking. Keeping it in the database also means the rule is not held only in Next.js (Constitution II), and the Go integration suite — which already runs against a real database in CI — can test it directly, including concurrency.
- **Alternatives considered**: Better Auth's rate limiter (keyed by IP and path only, the gap #99 exists to close); SQL in TypeScript (no database in the frontend's CI job, so the concurrency guarantee would go untested there); an endpoint on the Go service (Go does not take part in sign-in, and adding a hop for this would be a new architectural path without need).

## 2. The key

- **Decision**: HMAC-SHA256 of the address after trimming and lower-casing, keyed with `BETTER_AUTH_SECRET`, base64url.
- **Rationale**: FR-009. A plain SHA-256 of an email address is reversible by guessing — addresses are not secret, so anyone holding the table could test candidates. With the server's secret as the key that needs the secret too. Better Auth itself lower-cases before lookup; trimming as well makes " A@B.test" and "a@b.test" one address.
- **Alternatives considered**: storing the user id (only exists for registered addresses — that difference is what FR-006 forbids); plain SHA-256 (reversible, above).

## 3. Hooks

- **Decision**: `hooks.before` on `/sign-in/email` refuses a locked address with `APIError("TOO_MANY_REQUESTS")` and the fixed message; `hooks.after` reads `ctx.context.returned` and records a failure for a 401 `INVALID_EMAIL_OR_PASSWORD`, clears on success, and does nothing otherwise (403 `EMAIL_NOT_VERIFIED`, 429); `emailAndPassword.onPasswordReset` clears the account's address.
- **Rationale**: Better Auth 1.7.5 returns 401 `INVALID_EMAIL_OR_PASSWORD` for both an unknown address and a wrong password, so recording on that code treats both alike (FR-006). The unverified refusal is 403, so it is excluded (FR-010). `onPasswordReset` is called by `/reset-password` only after the password has been changed, not when a reset is requested (FR-005).
- **Alternatives considered**: wrapping the sign-in endpoint (no supported way); a plugin (more machinery than two hooks).

## 4. During a lock

- **Decision**: a locked attempt is refused before the password is checked, so it is neither counted nor able to extend the lock; once `locked_until` passes, the next failure starts the count at one.
- **Rationale**: spec edge cases. Refusing before the check also means a correct password during the lock does not reveal itself by a different response.

## 5. The message

- **Decision**: "Too many sign-in attempts. Try again in 15 minutes, or reset your password." — status 429, code `SIGN_IN_LOCKED`.
- **Rationale**: FR-007. The sign-in form already shows the server's message as-is, so no form change is needed. The same message for every address (FR-006).

## 6. Migration number

- **Decision**: `000013`.
- **Rationale**: `000011` and `000012` are taken by #57 and #58, not yet merged. Those must merge first: a database already at 13 would not apply 11 or 12.
