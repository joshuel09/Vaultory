# Contract: `POST /api/auth/sign-in/email`

Unchanged except one response.

| Situation | Status | Body `code` | Message |
|---|---|---|---|
| Address locked (5 consecutive failures in force) — whether or not it has an account | 429 | `SIGN_IN_LOCKED` | Too many sign-in attempts. Try again in 15 minutes, or reset your password. |
| Wrong password, or no account for the address | 401 | `INVALID_EMAIL_OR_PASSWORD` | (unchanged) |
| Address not verified | 403 | `EMAIL_NOT_VERIFIED` | (unchanged) |
| Per-network ceiling reached | 429 | (Better Auth's) | (unchanged) |

The 401 and 429 `SIGN_IN_LOCKED` responses are identical for an address with and without an account.
