# Research: Collector Authentication

All findings below were taken from Better Auth 1.7.5's own source in `node_modules`, not from
documentation prose. The documentation page for the database schema renders its tables through a
component that does not survive being fetched as text, so the package itself was installed into a
scratch directory and read.

---

## Decision 1 — How Go verifies a session

**Decision.** The Go service reads the session cookie, verifies its HMAC signature with the shared
secret, and then looks the token up in the `session` table, joining `user`. A request is
authenticated only if the signature verifies *and* a row exists *and* `expires_at` is in the
future. Go never calls the Next.js application and never reads an identity from a header, body, or
query parameter.

**What made this possible.** Better Auth's session lookup is a plain equality match on a column:

```js
findSession: async (token) => { ... adapter.findOne({ model: "session",
    where: [{ value: token, field: "token" }], join: { user: true } }) }
```

The token in the database is the token in the cookie — it is not hashed on the way in — so another
process holding the same database can perform the identical lookup. The cookie wrapping it is
(`better-call/dist/crypto.cjs`):

```js
export const signCookieValue = async (value, secret) => {
  const signature = await makeSignature(value, secret);   // HMAC-SHA256, then btoa()
  value = `${value}.${signature}`;
  return encodeURIComponent(value);
};
```

So the cookie is `urlencode(token + "." + base64(HMAC_SHA256(secret, token)))`. The signature is
standard base64 with padding, not base64url — a detail worth getting right once rather than
debugging twice.

**Why this satisfies FR-013.** "Independent verification" means Go reaches its own conclusion from
evidence it can check. It does: it recomputes the HMAC itself, and it confirms the session exists
and has not expired by querying PostgreSQL, which Principle II already names as the authoritative
store. Nothing in the chain requires trusting the presentation layer.

**Why this satisfies FR-011.** Signing out deletes the session row. The next request finds nothing
and is refused immediately. There is no window during which a revoked session still works.

**Alternatives considered.**

- *A self-contained JWT that Go verifies by signature alone.* No database read, so it scales
  better and Go needs no knowledge of Better Auth's tables. Rejected because a signed token stays
  valid until it expires: sign-out could not take effect immediately, which FR-011 requires. Adding
  a revocation list to fix that reintroduces the database read it was meant to avoid, with more
  moving parts.
- *Go calls a Next.js endpoint to validate each session.* Rejected on two counts. It makes the
  backend depend on the frontend being up, inverting the dependency the architecture is built on,
  and it adds a network hop to every single request.
- *Next.js verifies and forwards the collector id in a header.* Rejected outright. This is exactly
  what FR-013 forbids and what Principle IV calls out: "Client-provided ownership, user IDs, roles,
  permissions, prices, or trusted values MUST NOT be accepted without verification."

---

## Decision 2 — Mapping an account to a collector

**Decision.** Keep `collectors` exactly as feature 001 left it and add one column:
`user_id text UNIQUE NOT NULL REFERENCES "user"(id) ON DELETE CASCADE`. Go resolves a request by
joining `session → user → collectors`, and everything downstream continues to use
`collectors.id` — the uuid that every collectible, image, and submission already references.

**Rationale.** Better Auth generates `user.id` as a text identifier of its own choosing. Vaultory's
`collectors.id` is a uuid referenced by foreign keys throughout feature 001. Making one become the
other would mean rewriting those references, which is a large change to satisfy a small tidiness.
A link column costs one join and changes nothing that already works.

**Alternatives considered.**

- *Use `user.id` as the collector id.* Rejected: it ripples through every table, every query, and
  every test written in feature 001, to no benefit.
- *Store `collector_id` on the session as an additional field.* Rejected: the value would be
  written by the presentation layer, and Go would be trusting it. Deriving it from the join keeps
  the authority in the database.

---

## Decision 3 — Creating a collector when an account is created

**Decision.** A PostgreSQL `AFTER INSERT` trigger on `"user"` inserts the matching `collectors`
row, shipped in a golang-migrate migration.

**Rationale.** Registration happens inside Better Auth, in Next.js. Having Next.js also write to
`collectors` would put persistence of Vaultory's own domain table in the presentation layer, which
Principle II forbids. A trigger keeps the rule in the database — which Go owns through its
migrations — and makes it atomic with the account's creation, so an account without a vault cannot
exist even if registration fails halfway.

**Alternatives considered.**

- *Better Auth's `databaseHooks.user.create.after`.* Rejected: it is frontend code writing to a
  backend-owned table, and it is not atomic with the insert that triggered it.
- *Create the collector lazily on first authenticated request.* Rejected: it puts a write on a read
  path, and it means a signed-in collector momentarily has no vault.

---

## Decision 4 — One migration tool, not two

**Decision.** Better Auth's CLI generates the schema once; the SQL is transcribed by hand into a
reversible golang-migrate migration. Better Auth never runs migrations against this database.

**Rationale.** The constitution requires that "PostgreSQL schema changes MUST use version-
controlled and reproducible migrations", and this project already has exactly one mechanism for
that. Two tools writing to one schema, each with its own idea of what version the database is at,
is how a database ends up in a state neither can reconcile. Transcribing once is a small, bounded
cost paid at planning time.

**Consequence to accept.** Upgrading Better Auth later may change its expected schema, and nothing
will detect that automatically. The quickstart therefore includes a drift check: regenerate the
schema and diff it against the migration.

---

## Decision 5 — Password hashing

**Decision.** Use Better Auth's default, which is scrypt as shipped, and do not configure a custom
hasher.

**Rationale.** FR-005 asks for a deliberately slow, salted hash from which the original cannot be
recovered. Scrypt is memory-hard and meets that. Substituting a different algorithm would mean
owning parameter choices and an upgrade path for no stated benefit. Go never sees a password or a
hash — it only ever reads sessions — so the hash never crosses the architectural seam.

---

## Decision 6 — Sliding expiry with a hard cap

**Decision.** Configure Better Auth's session with `expiresIn` of 30 days and `updateAge` so the
expiry is extended on use. The 90-day cap is enforced by Go at verification time, by comparing the
session's `created_at` against the cap.

**Rationale.** Better Auth refreshes `expires_at` as a session is used, which gives the sliding
window in FR-010 directly. It has no native notion of an absolute ceiling, so the cap has to live
somewhere. Go is the right place: it is already reading the row, `created_at` is already on it, and
putting the check where verification happens means the cap holds even if the frontend is
misconfigured.

**Alternative considered.** A scheduled job deleting sessions older than 90 days. Rejected as a
sole mechanism: a request arriving before the job runs would still be honoured. It remains useful
as housekeeping, and is out of scope here.

---

## Decision 7 — Where rate limiting lives

**Decision.** Sign-in rate limiting (FR-027) is enforced by Better Auth in the Next.js layer.

**Rationale.** Go never sees a sign-in attempt — authentication happens entirely in front of it —
so it cannot count failures it is not shown. This is a consequence of the chosen architecture
rather than a gap. It is worth stating plainly in the Constitution Check rather than leaving a
reader to notice that an enforcement rule lives in the frontend.

**What keeps this honest.** The rule being enforced is about *authentication attempts*, not about
who may see a vault. Authorization — the rule that actually protects collector data — remains
entirely in Go, on every request, unchanged.

---

## Decision 8 — Removing the development resolver

**Decision.** Delete `internal/identity/dev.go` and its production counterpart, and replace the
resolver with one that verifies Better Auth sessions. The build tag split goes away with them.

**Rationale.** The build tag existed because a session-minting endpoint had to be guaranteed absent
from production images. Once the resolver is real, there is nothing to exclude, and keeping a tag
that no longer guards anything is worse than removing it: it implies a protection that is no longer
being provided. FR-020 is then satisfied because no such mechanism exists in any build.

**What must be re-verified.** `make test` currently runs `go test -tags production` specifically to
assert the dev resolver is absent. That assertion becomes vacuous and must be replaced by one that
still means something — that no code path issues a session without verifying a credential.

---

## Decision 9 — Known dependency conflict

**Finding, not yet a decision.** Installing `better-auth` into `frontend/` fails today:

```
npm error Conflicting peer dependency: vite@6.4.3
Found: vite@5.4.21  (required by vitest@2.1.9 and @vitest/mocker)
```

The frontend pins `vitest@^2.1.8`, which brings vite 5. Something in Better Auth's tree wants
vite 6. Options are to upgrade vitest to 3.x, which uses vite 6, or to install with
`--legacy-peer-deps`, which suppresses the report without resolving anything.

**Recommendation.** Upgrade vitest. It is a test-only dependency, the 33 existing frontend tests
are the blast radius, and `--legacy-peer-deps` leaves a known-inconsistent tree that the next
person has to rediscover. This is called out as its own task rather than absorbed silently into
another one, because it can fail independently of anything to do with authentication.
