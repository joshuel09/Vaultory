# Quickstart: Collector Authentication

Walkthroughs that demonstrate the feature and, where it matters, prove a property rather than
assert it. Each names the requirements it covers.

## Prerequisites

```bash
make up          # database, migrations, backend, frontend
```

There is no development sign-in any more. `VAULTORY_DEV_IDENTITY` does nothing, and
`/api/dev/session` returns 404 — that is the point of the feature.

---

## Walkthrough A — Register and get a vault

Covers US1, FR-001, FR-002, FR-004, SC-001.

1. Open <http://localhost:3000>. The landing page's call to action now leads to registration
   (FR-024).
2. Register with an email and a password of at least 12 characters.
3. Expect to land in an **empty** vault.

```bash
docker compose --profile dev exec -T postgres psql -U vaultory -d vaultory_dev -At -c \
  'select count(*) from "user"; select count(*) from collectors;'
# Expect 1 and 1 — the trigger created the collector (Decision 3)
```

Then try the failure cases and expect each to be refused, with the reason named against the field:
the same email again (FR-002), the same email in a different case (FR-003), a password of 11
characters (FR-004).

---

## Walkthrough B — Sign out, sign in, stay signed in

Covers US2, US3, FR-007, FR-009, FR-011, SC-002, SC-003.

Sign out, then confirm the old session is dead rather than merely forgotten. Capture the cookie
before signing out and replay it afterwards:

```bash
# with the cookie captured while signed in
curl -s -o /dev/null -w "%{http_code}\n" -b "better-auth.session_token=<captured>" \
  http://localhost:8080/api/collectibles
# Expect 401 — the row is gone, so there is no window where a revoked session still works
```

Sign back in, close the browser, reopen it: still signed in.

---

## Walkthrough C — The boundary this feature exists to protect

Covers FR-013, FR-014, SC-004, SC-005. **This is the walkthrough to run if you only run one.**

Every request below goes straight to the Go service on :8080, bypassing Next.js entirely. Each must
be refused.

```bash
B=http://localhost:8080/api/collectibles

# 1. No session at all
curl -s -o /dev/null -w "no cookie          -> %{http_code}\n" $B

# 2. An identity asserted in a header, with no session (FR-013)
curl -s -o /dev/null -w "forged header      -> %{http_code}\n" \
  -H "X-Collector-Id: 11111111-1111-4111-8111-111111111111" $B

# 3. An identity asserted in a query parameter
curl -s -o /dev/null -w "forged query       -> %{http_code}\n" \
  "$B?collector_id=11111111-1111-4111-8111-111111111111"

# 4. A real token with the signature stripped
curl -s -o /dev/null -w "unsigned token     -> %{http_code}\n" -b "better-auth.session_token=<token>" $B

# 5. A real token with a corrupted signature
curl -s -o /dev/null -w "bad signature      -> %{http_code}\n" \
  -b "better-auth.session_token=<token>.AAAA..." $B
```

Expect `401` five times. Anything else is a privacy failure, not a bug to triage later.

Then the two-collector check, which is feature 001's guarantee still standing: register a second
account, note the first collector's collectible id, and request it while signed in as the second.
Expect **404, never 403** — a 403 confirms the thing exists.

---

## Walkthrough D — Session expiry, both halves

Covers FR-010, SC-003.

The sliding window and the 90-day cap cannot be waited out, so move the row instead:

```bash
PSQL='docker compose --profile dev exec -T postgres psql -U vaultory -d vaultory_dev -c'

# Expired: should stop working immediately
$PSQL "update session set \"expiresAt\" = now() - interval '1 minute'"
# request -> 401

# Sign in again, then age the session past the cap while leaving expiresAt healthy.
# This is the half Better Auth does not enforce — if it passes, Go is not applying the cap.
$PSQL "update session set \"createdAt\" = now() - interval '91 days',
                          \"expiresAt\" = now() + interval '20 days'"
# request -> 401
```

The second case is the one worth being deliberate about. It is the only check that proves the cap
is enforced where Decision 6 says it is.

---

## Walkthrough E — Signed-out navigation

Covers US3, FR-022, FR-022a, SC-010, SC-013.

Signed out, open `/collection/new`. Expect to arrive at the sign-in page, and after signing in to
land back on `/collection/new` rather than the collection root.

Then the open redirect, which is the part a reviewer should insist on seeing:

```
/sign-in?next=https://example.com        -> after sign-in, NOT sent to example.com
/sign-in?next=//example.com              -> likewise
/sign-in?next=/collection                -> sent to /collection
```

A destination taken from a request and followed after sign-in is an open redirect unless it is
constrained to Vaultory's own paths. FR-022a exists because this walkthrough is easy to pass
accidentally.

---

## Walkthrough F — No fixtures left behind

Covers FR-017, FR-017a, SC-007.

```bash
docker compose --profile dev exec -T postgres psql -U vaultory -d vaultory_dev -At -c \
  "select count(*) from collectors where user_id is null;
   select count(*) from collectibles c
     where not exists (select 1 from collectors x where x.id = c.collector_id);"
# Expect 0 and 0 — no accountless collector, no orphaned collectible
```

Then see the restriction for yourself, connecting as the role the frontend uses rather than the
backend's:

```bash
docker compose --profile dev exec -T postgres \
  psql -U vaultory_auth -d vaultory_dev -c 'select count(*) from collectibles;'
# Expect: ERROR: permission denied for table collectibles

docker compose --profile dev exec -T postgres \
  psql -U vaultory_auth -d vaultory_dev -c 'select count(*) from "session";'
# Expect: a number — its own tables are readable
```

That refusal is what keeps the plan's claim honest once Next.js holds a database connection: a bug
in the frontend cannot read a collection, because PostgreSQL will not let it.

Then roll the migration back and confirm the seeded fixtures return, because Principle IV requires
migrations to be reversible and a down migration nobody has run is a guess.

---

## Walkthrough G — Production actually starts

Covers FR-018, FR-019, FR-020, SC-008, SC-009.

```bash
make prod-build
docker compose -f compose.prod.yaml up
```

The backend refused to start throughout features 002 and 003 because it had no way to identify
anybody. With a real resolver it must now start and serve traffic (FR-019, SC-009).

Then prove the absence rather than assert it, the way feature 002 did:

```bash
cid=$(docker create vaultory-prod-backend); docker cp $cid:/usr/local/bin/vaultory-api /tmp/b
docker rm -f $cid
strings /tmp/b | grep -ci "dev.session\|DevResolver\|VAULTORY_DEV_IDENTITY"   # expect 0
```

`grep -c` on the shipped binary is the check. A test asserting "the dev resolver is gone" passes
trivially once the file is deleted; looking in the artifact is what makes SC-008 mean something.

---

## Walkthrough H — Rate limiting

Covers FR-027, SC-012.

Eleven wrong passwords for one account inside fifteen minutes. The eleventh is refused with a
message saying when to try again; the account is not locked permanently. Wait, or clear the
`rateLimit` rows, and the same account signs in normally.

---

## Walkthrough I — Schema drift

Covers Decision 4.

Better Auth's schema is transcribed by hand into a golang-migrate migration, so nothing detects it
if a library upgrade changes what the library expects.

```bash
cd frontend && npx @better-auth/cli generate --output /tmp/ba-schema.sql
# diff against backend/migrations/000007_*.up.sql — differences are the upgrade's real cost
```

Run this when upgrading Better Auth, not on a schedule.

---

## Results, 2026-09-21

Walked on a real stack, against the merged feature. Recorded rather than assumed.

| Walkthrough | Result |
|---|---|
| A — register and get a vault | **pass** — registration 200, trigger created one collector |
| B — sign out, sign in, stay signed in | **pass** — a captured cookie is refused by Go immediately after sign-out |
| C — the boundary | **pass** — automated as T019/T020/T020b; seven asserted identities ignored, ten session shapes refused |
| D — session expiry, both halves | **pass** — automated as T021; 91 days old with a healthy `expiresAt` is refused |
| E — signed-out navigation | **pass** — automated in `navigation.spec.ts` and `safe-redirect.test.ts` |
| F — no fixtures left behind | **pass** — up/down/up clean, fixtures restored by the down migration |
| G — production actually starts | **pass** — see the table above; first successful start since feature 002 |
| H — rate limiting | **partial** — the sign-in rule is configured and asserted in `auth-config.test.ts`; the eleventh failed attempt was not walked by hand |
| I — schema drift | **pass** — the schema was regenerated during T005 and transcribed column-by-column |

Two things this exercise found rather than confirmed, both fixed:

- `make prod-build` could not build. Feature 004 added two required variables and the target still
  supplied only feature 002's two. The second time that target has broken this way, so `make
  check` now covers the full set.
- `Field` pointed `aria-describedby` at an element that was not rendered whenever a field carried
  both a hint and an error. Latent since feature 001; the register page is the first field
  anywhere with a hint, a required marker and an error at once.

## Acceptance summary

| Walkthrough | Covers |
|---|---|
| A | US1 · FR-001..004 · SC-001 |
| B | US2, US3 · FR-007, FR-009, FR-011 · SC-002, SC-003 |
| C | **FR-013, FR-014, FR-015** · SC-004, SC-005 |
| D | FR-010 · SC-003 |
| E | FR-022, FR-022a · SC-010, SC-013 |
| F | FR-017, FR-017a · SC-007 |
| G | FR-018..020 · SC-008, SC-009 |
| H | FR-027 · SC-012 |
| I | Decision 4 |

SC-006 — that a wrong password and an unknown email are indistinguishable — is not a walkthrough
because eyeballing two responses is not evidence. It belongs in a contract test that compares
status, body, and timing.
