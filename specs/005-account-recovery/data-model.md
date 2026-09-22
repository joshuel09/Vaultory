# Data Model: Account Recovery

No new tables, no migration to any Vaultory table, and no change to `collectors`. What changes is
what goes *into* one column of a table feature 004 already created.

## Changed: what `verification` stores

Better Auth's `verification` table exists from feature 004 and was unused — email verification was
out of scope. It now holds outstanding password resets.

| Column | Holds | Note |
|---|---|---|
| `identifier` | `reset-password:<token>` | **Stored hashed.** SHA-256, base64url |
| `value` | the account id the reset is for | |
| `expiresAt` | one hour after issue | Better Auth's default, and FR-010 |
| `createdAt` / `updatedAt` | | |

The change is in `identifier`, and it is the whole of FR-015. By default Better Auth writes the
token itself there, so a copy of this table would be a set of working reset links. Hashed, the
stored value is a digest: it can confirm a token presented to it and cannot produce one.

A reset is spent by `consumeVerificationValue`, which deletes the row as it reads it. Single use is
therefore a property of the storage rather than a check that could be forgotten.

## Not stored at all: verification tokens

Email verification tokens are signed JWTs. Nothing is written when one is issued and nothing is
read when one is followed — the signature and the expiry inside the token are the whole check.

This is why FR-004 and FR-006 narrow to reset tokens (research Decision 3). There is no row to
spend and none to invalidate. The consequence is small and specific: a verification link can be
replayed until it expires, and replaying it sets `emailVerified` to true on an account where it is
already true.

## Changed: `user.emailVerified`

Already present from feature 004, always `false`. It now means something:

| Transition | Cause |
|---|---|
| `false → true` | a valid verification link is followed |
| `false → true` | a password reset completes (FR-014a) |
| `true → false` | never, in this feature. Changing an address is out of scope |

## How a reset resolves

```
request  ──> address looked up
             │
             ├── no account: send nothing, answer identically (FR-008)
             │
             └── account: token generated, SHA-256 of it stored with the account id,
                          the token itself mailed and never persisted
                                    │
complete ──> token presented ───────┘
             │
             ├── consumeVerificationValue deletes the row as it reads (single use)
             ├── expired or absent -> refused, indistinguishably
             ├── password replaced
             ├── every session for that account deleted (FR-018)
             └── emailVerified set true if it was not (FR-014a)
```

The session deletion is what makes a reset an eviction rather than a rename. It needs nothing from
the Go service: the resolver already requires a live session row on every request, so deleting the
rows ends those sessions at the next request with no notification, no cache, and no window.

## What deliberately does not change

`collectors`, `collectibles`, `collectible_images`, `collectible_submissions` — untouched. The
restricted `vaultory_auth` role still has no privilege on any of them, and recovery needs none: it
reads and writes `user`, `verification`, `session` and `rateLimit`, all of which it already has.

That is worth stating because the instinct on a feature like this is to reach for the backend. Go
verifies sessions and scopes queries by `collector_id`, exactly as before, and none of it needed
revisiting.
