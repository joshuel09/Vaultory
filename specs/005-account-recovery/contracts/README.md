# Contracts: Account Recovery

## The Go API contract does not change

Not one path, request shape, or response shape in
`specs/001-add-browse-collectibles/contracts/openapi.yaml` changes. A task verifies the generated
frontend types are byte-identical afterwards, the same check feature 004 ran.

Recovery is a credentials-and-sessions concern, and feature 004 placed those in Better Auth. The
Go service's role is unchanged: verify the session on every request, scope every query by
`collector_id`.

## Better Auth's recovery routes stay out of the contract

`/api/auth/forget-password`, `/api/auth/reset-password`, `/api/auth/verify-email`,
`/api/auth/send-verification-email` are served by Next.js and consumed only by Vaultory's own
pages. They do not cross the frontend/backend seam, so Constitution III does not reach them —
writing them into `openapi.yaml` would assert that Go serves them, which is false.

## The contract that matters here is what a token is

Two kinds, with deliberately different properties. Confusing them is the mistake this page exists
to prevent.

| | Verification | Reset |
|---|---|---|
| Form | signed JWT, self-contained | random token |
| Stored | **nothing** | SHA-256 of the token, in `verification.identifier` |
| Lifetime | 24 hours | **1 hour** |
| Single use | **no** — nothing records a spend | **yes** — the row is deleted as it is read |
| Grants a session | **no** (FR-002a) | yes, on completion |
| Worst case if leaked | an address is marked verified | **account takeover** |

The asymmetry is the design. A reset link is a way into a vault, so it is short-lived, spent on
use, and never stored in a form that could be replayed. A verification link proves only that
somebody reads a mailbox, so it can be stateless without consequence.

## Rules that hold for both

- **Never logged.** Not the token, not the full link, not in an error, not in development. A link
  printed to a log is a credential in a log (FR-016).
- **Refusals are indistinguishable.** Expired, already spent, altered, or issued for another
  account all answer the same way. A caller learns nothing about which part of their attempt was
  wrong (FR-017).
- **Requesting reveals nothing.** A reset request for an address with no account answers exactly as
  one for an address with an account — status, body, and timing (FR-008).

## The mail interface

One function, so the transport is swappable and the call sites never learn which one is in use:

```
send({ to, subject, text }) -> Promise<void>
```

Development points it at Mailpit over SMTP, which holds messages in memory and has no outbound
path. Production points it at a real service. Nothing else differs, which is what makes the
development path worth trusting: it exercises the same code.

## What must not be reintroduced

A recovery flow must not establish a session by any route the Go service does not already verify
(FR-025). This feature adds two new ways to become authenticated, which is exactly when feature
004's guarantee is most likely to be undone by accident — a "helpful" auto sign-in after
verification would do it, and is specifically turned off (research Decision 4).
