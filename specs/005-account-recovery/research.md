# Research: Account Recovery

Every finding below was read from Better Auth 1.7.5's own source in `node_modules`, not from
documentation prose. Three of them change what the spec can promise, and one of those is the
reason this feature needs a deviation recorded rather than a straightforward implementation.

---

## Decision 1 — Reset tokens are stored hashed

**Decision.** Set `verification.storeIdentifier: 'hashed'`.

**What made this necessary.** By default Better Auth stores a reset token *as the token*:

```js
await ctx.context.internalAdapter.createVerificationValue({
  identifier: `reset-password:${verificationToken}`,   // dist/api/routes/password.mjs:77
  ...
})
```

Anyone who can read the `verification` table — a backup, a replica, an injection — could take over
any account with an outstanding reset, and nothing about the running system would look wrong. That
is precisely what FR-015 forbids.

**What makes it possible.** `dist/db/verification-token-storage.mjs` accepts `'plain'` (the
default), `'hashed'` (SHA-256, base64url), or a custom `{ hash }`. With `'hashed'`, the stored
value is a digest and the token is not recoverable from it.

**On the absence of a salt.** SHA-256 without one is right here and would be wrong for a password.
A password is low-entropy and guessable, so a salt defeats precomputation. A reset token is many
bytes of randomness, so there is nothing to precompute against. Adding a salt would buy nothing and
imply the token needed stretching, which it does not.

**Alternative considered.** Leave it plain and restrict who can read the table. Rejected: the
restricted role must read `verification` to do its job, so the restriction would protect nothing,
and it makes a database copy equivalent to account takeover.

---

## Decision 2 — Sessions are revoked on reset, explicitly

**Decision.** Set `emailAndPassword.revokeSessionsOnPasswordReset: true`.

**Rationale.** It is **off by default** (`dist/api/routes/password.mjs:171`). Left alone, a reset
changes the password and leaves every existing session working — which is the failure FR-018 names:
it changes the key without changing the lock. Someone resetting because they fear another person
has their password would not evict them.

**Why nothing is needed in Go.** Revocation deletes the session rows, and the Go resolver already
requires a live row on every request. The eviction is immediate and free: the backend needs no
change and no notification, because it was never trusting anything but the database.

---

## Decision 3 — Verification tokens are stateless, and two requirements must narrow

**Finding.** Email verification does not store anything. The token is a signed JWT:

```js
async function createEmailVerificationToken(secret, email, updateTo, expiresIn = 3600, extra) {
  return await signJWT({ email: email.toLowerCase(), ... }, secret, expiresIn)
}
```

Two consequences follow, and neither can be configured away:

- **FR-004 (single use) cannot hold for verification.** Nothing records that a JWT was spent, so it
  works until it expires.
- **FR-006 (a new link invalidates the previous) cannot hold either.** Old links stay valid for
  their lifetime.

**Decision.** Narrow both requirements to *reset* tokens, and say why in the spec rather than
letting them quietly fail. Reset tokens are stored and are already single-use —
`consumeVerificationValue` (`password.mjs:157`) deletes as it reads — so FR-011 and FR-012 hold
where they matter.

**Why this is acceptable for verification specifically.** A verification link grants no access
(FR-002a), and all it does is set a boolean that is already true on a second use. Replaying one
re-verifies an address the holder has already proven they control. The consequence of a replayed
*reset* link would be account takeover; the consequence of a replayed *verification* link is
nothing.

**Alternatives considered.**

- *Record spent verification tokens in a denylist.* Buys single-use at the cost of a table, a
  cleanup job, and a second source of truth — to prevent an action with no effect.
- *Bump a per-user counter into the token and reject stale ones.* Same cost, same non-benefit, and
  it invents a mechanism the library would fight on upgrade.

**This is recorded in Complexity Tracking, and the spec is amended rather than left aspirational.**
A requirement that cannot hold is worse than one that was never written: it reads as a guarantee.

---

## Decision 4 — Verification does not sign anyone in

**Decision.** Leave `emailVerification.autoSignInAfterVerification` unset.

**Rationale.** It is opt-in (`email-verification.mjs:297`), and FR-002a wants it off. Verification
links are usually opened on a phone or another browser; turning one into a session would make a
24-hour-lived email a bearer credential for a vault, so a forwarded or archived message would be
access to a collection. The `createSession` calls elsewhere in that file are in the change-email
branches, which this feature does not use.

---

## Decision 5 — Link lifetimes

**Decision.** Reset: 1 hour, which is already the default (`password.mjs:73`). Verification: 24
hours, set explicitly — the default is 1 hour and FR-003 allows a day.

**Rationale.** The asymmetry is the point and is inherited from the spec: a reset link is worth far
more to an attacker than a verification link, so it lives for less time. Twenty-four hours for
verification accommodates somebody who registers at night and reads their mail the next morning.

---

## Decision 6 — Mail: a real provider in production, a local catcher in development

**Decision.** Send through `nodemailer` behind one small interface. In development, Compose runs
**Mailpit**, which accepts SMTP on 1025 and serves every captured message on a web interface at
8025. Production points the same interface at a real SMTP service.

**Rationale.** FR-023 asks that a developer read every message with no third-party account and no
mail leaving the machine. Mailpit is a single container, holds messages in memory, and has no
outbound path at all — a message cannot escape by misconfiguration because there is nowhere for it
to go.

**Alternatives considered.**

- *A hosted provider with a sandbox mode.* Requires an account before a clone can run, which the
  project has avoided everywhere else — `make up` is meant to need Docker and nothing else.
- *Log the link to stdout in development.* Simple, and it fails FR-016: a link printed to a log is
  a credential in a log, and the habit outlives the convenience.

---

## Decision 7 — Rate limiting the recovery paths

**Decision.** Better Auth `customRules` on the recovery endpoints: three per hour each, with
verification and reset counted separately, matching FR-020.

**Rationale.** This is the same mechanism feature 004 used for sign-in, and the same lesson applies:
the global ceiling does not override a path's own rule, and Better Auth applies stricter built-in
defaults to some paths. Each recovery path is therefore stated explicitly rather than assumed to
inherit anything.

---

## Decision 8 — Nothing in the Go service changes

**Decision.** No backend change, no migration to `collectors`, no new endpoint.

**Rationale.** Worth stating plainly because it is easy to assume otherwise. Recovery is entirely a
credentials-and-sessions concern, which feature 004 placed in Better Auth. Go's job — verify the
session, scope every query by `collector_id` — is untouched, and session revocation reaches it for
free because it already requires a live row.

The one schema change is Better Auth's own: `verification` gains hashed identifiers, which is a
change of content rather than shape. The `rateLimit` table already exists from feature 004.
