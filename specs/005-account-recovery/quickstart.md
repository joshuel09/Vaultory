# Quickstart: Account Recovery

Walkthroughs that demonstrate the feature, and in three places prove a property rather than assert
it. Each names the requirements it covers.

## Prerequisites

```bash
make up
```

Mailpit joins the stack. Every message Vaultory sends locally is captured there and readable at
<http://localhost:8025>; none of it leaves the machine.

---

## Walkthrough A — Verify an address

Covers US1, FR-001, FR-002, FR-003, SC-002.

1. Register at <http://localhost:3000/register>.
2. Your vault opens, with a banner saying the address is unverified.
3. Open <http://localhost:8025>. The verification message is there.
4. Follow its link. The address is verified and the banner is gone.

```bash
# The account's state, before and after
docker compose --profile dev exec -T postgres psql -U vaultory -d vaultory_dev -At \
  -c 'select "email", "emailVerified" from "user" order by "createdAt" desc limit 1;'
```

---

## Walkthrough B — A verification link is not a way in

Covers **FR-002a**, SC-013. Short, and the one most easily broken by a later convenience.

Open the same link in a private window, or on a phone — anywhere with no session.

Expect: the address is verified, and you are **offered sign-in** rather than signed in. No
collection content appears.

A verification link lives for 24 hours. If it granted a session, a forwarded or archived message
would be a way into somebody's vault for a day. Proving you can read a mailbox is not proving you
know a password.

---

## Walkthrough C — Get back in after forgetting the password

Covers US2, FR-007, FR-009, FR-010, FR-013, FR-014, SC-001.

1. Sign out. Choose "forgotten your password" on the sign-in page.
2. Enter your address. Read the message in Mailpit and follow its link.
3. Choose a new password. You arrive in your own collection.
4. Sign out and confirm the **old** password is refused.

Then the failures, each of which must be refused without consuming anything it should not:

```
a password under 12 characters  -> refused, link still usable
the same link a second time     -> refused (the row was deleted when it was spent)
a link older than an hour       -> refused
a link with one character changed -> refused, identically to all of the above
```

---

## Walkthrough D — A reset is an eviction

Covers US3, **FR-018**, SC-005. **Run this one if you run only one.**

1. Sign in on two browsers — a normal window and a private one.
2. Confirm both reach the collection.
3. Reset the password from the first.
4. Reload the second.

Expect the second to be signed out and to show no collection content.

```bash
# Capture the second browser's cookie before the reset, then afterwards:
curl -s -o /dev/null -w "%{http_code}\n" \
  -H "Cookie: better-auth.session_token=<captured>" http://localhost:8080/api/collectibles
# Expect 401 — the session row is gone
```

This is what separates a reset from a rename. A reset that leaves other sessions working does not
do the thing people reset passwords for.

---

## Walkthrough E — Requesting a reset reveals nothing

Covers FR-008, SC-003.

```bash
for a in "someone-who-exists@example.test" "nobody-at-all@example.test"; do
  curl -s -o /dev/null -w "$a -> %{http_code} %{time_total}s\n" \
    -X POST http://localhost:3000/api/auth/request-password-reset \
    -H 'Content-Type: application/json' -H 'Origin: http://localhost:3000' \
    -d "{\"email\":\"$a\",\"redirectTo\":\"/reset-password\"}"
done
```

Expect the same status and comparable timing. The bodies must match too. Anything that differs
turns this form into a way to discover who has a vault here.

---

## Walkthrough F — The stored token is not the token

Covers **FR-015**, SC-009. This is the check that distinguishes a configured system from one that
merely looks configured.

Request a reset, then read what was stored:

```bash
docker compose --profile dev exec -T postgres psql -U vaultory -d vaultory_dev -At \
  -c 'select identifier from verification order by "createdAt" desc limit 1;'
```

Expect a digest — **not** a value containing the token from the link in Mailpit. Observed
2026-09-25:

```
stored:     rVNC4Ap_ucStrqbCTF5A2SNkToyPTh-sDvwXLs5ZlTA
link token: 49EYPuVZUYL6BJ50EJmDHlPl
``` Then prove the
stored form is useless on its own:

```
take the identifier from the database, use it as the token in the reset link -> refused
```

If the stored value works as a token, a copy of this table is a set of working keys to every
account with an outstanding reset, and nothing about the running system would look wrong.

---

## Walkthrough G — No token reaches a log

Covers FR-016, SC-007.

```bash
make logs > /tmp/vaultory.log 2>&1 &   # then exercise walkthroughs A and C
grep -cE "reset-password:|verify-email\?token=|[A-Za-z0-9_-]{32,}" /tmp/vaultory.log
```

Expect no token in Vaultory's own log lines — the `{"event":"auth",…}` entries. Recovery events
are recorded there (requested, sent, completed, refused) with none of their contents (FR-021).

**Observed 2026-09-26, and fixed.** The logger was putting tokens into the log itself: the action
was taken straight from the path, so `/api/auth/reset-password/:token` became
`"action":"reset-password/RWDXKO80ywIDmFMemzr2pXCd"`. Forty-five of them were in one run.
`lib/auth-logging.ts` now keeps only lowercase word segments, and a unit test pins it.

**One source remains, and it is not ours.** The Next.js development server logs every request URL,
including `GET /reset-password?…&token=…`. That is development-only — a production build does not
log request URLs — and it is worth being plain about the deeper point: a token carried in a URL is
exposed to browser history, referrers, and any proxy along the way, whoever is logging. That is
precisely why a reset link lives one hour and works once, rather than why it is safe.

---

## Walkthrough H — Rate limiting

Covers FR-020, SC-011.

Request a verification message four times within an hour for one address. The fourth is refused,
with a message saying when to try again.

Then, in the same hour, request a **password reset** for that address: it must succeed. The two
limits are counted separately precisely so that exhausting one cannot block the other at the moment
it is most needed.

---

## Walkthrough I — Mail stays on the machine

Covers FR-023, SC-008.

Mailpit holds messages in memory and has no outbound path — there is nowhere for a message to
escape to, so it cannot leave by misconfiguration. Confirm the container publishes only its SMTP
and web ports, and that every message from walkthroughs A and C is readable at
<http://localhost:8025>.

---

## Results, 2026-09-26

Walked against the running stack. Recorded, not assumed.

| Walkthrough | Result |
|---|---|
| A — verify an address | **pass** — a real message arrives; the vault says unverified and offers a resend |
| B — a verification link is not a way in | **pass** — zero session cookies, `/collection` still unreachable |
| C — get back in after forgetting | **pass** — and the old password is refused afterwards |
| D — a reset is an eviction | **pass** — the other device is redirected to sign-in, its cookie refused by Go |
| E — requesting reveals nothing | **pass** — identical status and body for an address with an account and one without |
| F — the stored token is not the token | **pass** — `rVNC4Ap_…` stored against `49EYPuVZ…` in the link |
| G — no token reaches a log | **fixed, then pass** — see below |
| H — rate limiting | **pass** — the fourth request is refused while a reset for the same address still succeeds |
| I — mail stays on the machine | **pass** — Mailpit holds everything, publishes only SMTP and its web interface |

### What walking them found

**Walkthrough G failed the first time, and the leak was ours.** The auth logger took its action
straight from the path, so `/api/auth/reset-password/:token` was logged as
`"action":"reset-password/RWDXKO80ywIDmFMemzr2pXCd"` — forty-five tokens in one run. Fixed by
keeping only lowercase word segments, as a whitelist rather than a redaction list, so the next
dynamic segment is dropped without anyone remembering to. Three unit tests pin it.

**Walkthrough D found a second one.** The middleware only checks that a session cookie is
*present*, so a device whose session had been revoked sailed past it and met "your collection could
not be loaded" — the exact misleading state FR-022 exists to remove. A 401 from the backend now
redirects to sign-in, and the error state means what it says again.

**Two corrections to these instructions themselves.** The endpoint is `/request-password-reset`,
not `/forget-password`; and walkthrough G's grep matched the Next development server's own request
log as well as ours, which is development-only and not something this feature controls.

## Acceptance summary

| Walkthrough | Covers |
|---|---|
| A | US1 · FR-001..003 · SC-002 |
| B | **FR-002a** · SC-013 |
| C | US2 · FR-007, FR-009..014 · SC-001, SC-006 |
| D | US3 · **FR-018** · SC-005 |
| E | FR-008 · SC-003 |
| F | **FR-015** · SC-009 |
| G | FR-016, FR-021 · SC-007 |
| H | FR-020 · SC-011 |
| I | FR-023 · SC-008 |

FR-014a — that completing a reset also verifies the address — is checked at the end of walkthrough
C rather than given its own section: after resetting, the unverified banner must be gone.

SC-004, that every refusal is indistinguishable, is not a walkthrough. Comparing four error
messages by eye is not evidence; it belongs in a test that compares status, body and timing.
