# Implementation Plan: Collector Authentication

**Branch**: `15-collector-authentication` · **Spec**: [spec.md](spec.md) · **Issue**: #15
**Date**: 2026-09-19 · **Status**: planned

## Summary

Better Auth runs in Next.js and owns registration, sign-in, password hashing, and session issuance.
The Go service verifies every session itself — recomputing the cookie's HMAC with the shared secret
and confirming the session row exists, has not expired, and is within a 90-day cap — and remains
the only thing that authorizes access to a vault.

The development resolver is deleted. A production image starts for the first time.

## Technical Context

| | |
|---|---|
| **Frontend** | Next.js 15.5 App Router, TypeScript strict, Tailwind, shadcn/ui |
| **Auth library** | Better Auth 1.7.5, email + password, no social providers |
| **Backend** | Go 1.26, `net/http`, pgx v5 |
| **Database** | PostgreSQL 17; golang-migrate, one tool only (research Decision 4) |
| **Session seam** | Signed cookie + database lookup ([contracts/README.md](contracts/README.md)) |
| **Testing** | Go unit/integration/contract; Vitest; Playwright |
| **Unknowns** | None. All nine research decisions are resolved from Better Auth's own source |

## Constitution Check

| Principle | Assessment |
|---|---|
| **I. Collector-First Experience** | Registration and sign-in report every problem at once, against the field responsible, announced to assistive technology, and never discard what was typed (FR-026). The misleading 401 becomes a sign-in invitation (FR-022). Dark mode is first-class, from the same tokens as every other page. |
| **II. Separation of Responsibilities** | **The principle to watch in this feature.** See the two entries below. Authorization stays wholly in Go: every collection query remains scoped by `collector_id`, unchanged from feature 001. |
| **III. Contract-First** | The OpenAPI contract does not change, and a task verifies the generated types are byte-identical. Better Auth's routes are deliberately excluded: they do not cross the frontend/backend seam. The session *is* a contract between two programs, so it is written down in `contracts/README.md` where OpenAPI cannot express it. |
| **IV. Data Integrity and Security** | Go verifies rather than trusts (FR-013), which is this principle's "client-provided user IDs MUST NOT be accepted without verification" stated as a requirement. Schema changes are one reversible migration. Secrets stay out of source control. Security-sensitive operations fail closed: anything unverifiable is 401 with no content. |
| **V. Quality, Simplicity, Spec-Driven** | Work began from a specification with 30 requirements and 13 success criteria, clarified in four questions. Every walkthrough traces to requirements. |

### Where authentication lives in Next.js, and why that is not a violation

Principle II says business rules must not exist exclusively in the frontend, and that
"authorization, authoritative validation, persistence rules, and domain behavior MUST be enforced
by the Go backend."

Better Auth places *authentication* — proving who someone is — in Next.js. **Authorization** —
deciding what they may touch — stays entirely in Go, and Go does not take the frontend's word for
the first part either. It recomputes the signature and checks the row itself, which is why
Decision 1 rejected both the forwarded-header and the call-the-frontend designs.

The accepted cost: if Next.js is compromised, an attacker can mint sessions. That is true of any
design where something issues sessions. What this design preserves is that a *bug* in the frontend
— the far likelier failure — cannot cause one collector to see another's vault, because Go never
takes its word for an identity.

**That claim only holds if the frontend's database access is restricted, so it is.** Adopting
Better Auth means the Next.js process holds a PostgreSQL connection. Given the same credentials
Go uses, a frontend bug could read `collectibles` directly and every authorization check in Go
would be irrelevant — the sentence above would be false.

Better Auth therefore connects as a dedicated role with privileges on its own five tables and
nothing else. No `SELECT` on `collectibles`, `collectible_images`, `collectible_submissions`, or
`collectors`. The one thing it must do to a Vaultory table — create a collector when an account is
created — happens inside a `SECURITY DEFINER` trigger, so the privilege belongs to the trigger
rather than to the role that fired it.

The role is created without a password and provisioned from an environment variable, because a
credential in a tracked migration is what Principle IV forbids and this repository is public. A
`LOGIN` role with no password cannot authenticate, so forgetting to provision it stops the frontend
connecting rather than leaving an open account.

The restriction is enforced by PostgreSQL grants, not by convention, and a test asserts the role
is actually refused. Without it this feature would hand the presentation layer the keys to every
vault while the plan claimed the opposite.

### Rate limiting is enforced in Next.js

FR-027 is enforced by Better Auth, because Go never sees a sign-in attempt and cannot count
failures it is not shown. Recorded here rather than left for a reader to notice. It governs
authentication attempts, not access to collector data; the rule that actually protects a vault is
untouched.

## Project Structure

### Documentation

```
specs/004-collector-authentication/
├── spec.md              30 FRs, 13 SCs, 4 clarifications
├── plan.md              this file
├── research.md          9 decisions, from Better Auth's source
├── data-model.md        tables, the trigger, and how a request resolves
├── contracts/README.md  the session contract, which OpenAPI cannot express
├── quickstart.md        walkthroughs A–I
└── checklists/requirements.md
```

### Source

```
backend/
├── migrations/
│   ├── 000007_create_auth_tables.{up,down}.sql      Better Auth schema, transcribed
│   └── 000008_link_collectors_to_accounts.{up,down}.sql
│                                                     remove fixtures, add user_id, add trigger
├── internal/identity/
│   ├── identity.go                                   Resolver interface — unchanged
│   ├── session.go                                    NEW: verifies Better Auth sessions
│   ├── dev.go                                        DELETED
│   └── dev_production.go                             DELETED
└── internal/transport/httpapi/server.go               /api/dev/session removed

frontend/
├── lib/auth.ts                                        Better Auth server instance
├── lib/auth-client.ts                                 client hooks
├── app/api/auth/[...all]/route.ts                     Better Auth's own routes
├── app/(auth)/sign-in/page.tsx
├── app/(auth)/register/page.tsx
├── app/collection/…                                   401 → redirect to /sign-in?next=…
└── lib/safe-redirect.ts                               FR-022a
```

## Phases

| Phase | Contents |
|---|---|
| **0 — Foundation** | Upgrade vitest (Decision 9); transcribe the Better Auth schema; write both migrations |
| **1 — Go verification (US1/US2 backbone)** | `identity.Session` resolver; wire it in `main.go`; delete the dev resolver; integration and contract tests for walkthrough C |
| **2 — Frontend auth (US1, US2)** | Better Auth instance and routes; register and sign-in pages; landing CTA |
| **3 — Signed-out navigation (US3)** | Redirect with `next`, safe-redirect guard, the sign-in state replacing the 401 error |
| **4 — Production and verification** | Production image starts; binary inspected for absence; walkthroughs A–I |

Phase 1 can be built and tested before any of Phase 2 exists, by inserting a session row directly.
That ordering is deliberate: the security boundary gets its own tests before there is a UI to make
it look like it works.

## Risks

| Risk | Handling |
|---|---|
| **Signature encoding** — Better Auth uses standard base64; feature 001's dev cookie used base64url | Named explicitly in `contracts/README.md`. A unit test signs a known token and compares against a fixture generated by Better Auth itself, not by our own Go code |
| **Schema drift on upgrade** — the schema is transcribed by hand | Walkthrough I regenerates and diffs. Run on upgrade, not on a schedule |
| **The `-tags production` assertion goes vacuous** — it currently asserts the dev resolver is absent | Decision 8. Replaced by a check that no code path issues a session without verifying a credential, plus binary inspection in walkthrough G |
| **vitest upgrade breaks the 33 frontend tests** | Its own task, done first, so a failure is attributable to the upgrade rather than to authentication |
| **The 90-day cap is enforced in only one place** | Walkthrough D ages a session past the cap while leaving `expiresAt` healthy — a case Better Auth would let through |

## Complexity Tracking

| Deviation | Why needed | Simpler alternative rejected because |
|---|---|---|
| **Go reads tables it does not own** (`session`, `user`) | FR-013 requires independent verification and FR-011 requires sign-out to take effect immediately. Both are satisfied by one query | A self-contained JWT needs no shared tables, but stays valid until expiry, so sign-out could not be immediate. Adding revocation reintroduces the read with more parts |
| **Better Auth's camelCase columns need quoting in Go's queries** | Renaming them via Better Auth's `fields` options must be maintained against the library forever, and a mismatch fails at runtime rather than at migration time | Renaming looks tidier and hides a seam that is better left visible. The ugliness is confined to one resolver query |
| **A database trigger creates the collector row** | Registration happens in Next.js; having it write to `collectors` would put persistence of a backend-owned table in the presentation layer | A Better Auth `databaseHooks` callback is frontend code writing a backend table, and is not atomic with the insert that triggers it |
| **Next.js connects to PostgreSQL directly** | Better Auth owns the account and session tables and must read and write them. There is no way to adopt it without giving the Next.js process a database connection | Proxying Better Auth's queries through Go would mean Go implementing an interface it does not own, tracking it across upgrades, for no gain — Go would still not be the one deciding. **This access is restricted to a dedicated role** (see below), which is what keeps the constraint meaningful rather than nominal |

No other deviation.

## Post-Design Constitution Re-check

Re-evaluated after the artifacts above: **no new violations**. The three entries in Complexity
Tracking are the complete set, each with a stated reason and a rejected alternative.

The design makes one thing strictly better than before: `VAULTORY_DEV_IDENTITY` disappears, and
with it a documented, publicly described endpoint that issues a session to anyone who asks.
