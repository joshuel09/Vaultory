# Implementation Plan: Account Recovery

**Branch**: `24-account-recovery` · **Spec**: [spec.md](spec.md) · **Issue**: #24
**Date**: 2026-09-22 · **Status**: planned

## Summary

Better Auth gains email verification and password reset, configured so that three things hold which
are off or absent by default: reset tokens are stored hashed, a reset revokes every other session,
and following a verification link does not sign anyone in.

Mail goes through one small interface — Mailpit locally, a real service in production. **Nothing in
the Go service changes.**

## Technical Context

| | |
|---|---|
| **Frontend** | Next.js 15.5 App Router, TypeScript strict, Tailwind, shadcn/ui |
| **Auth** | Better Auth 1.7.5 — `emailVerification`, `emailAndPassword.sendResetPassword` |
| **Mail** | `nodemailer` behind one interface; Mailpit in Compose for development |
| **Backend** | Go 1.26 — **unchanged** |
| **Database** | PostgreSQL 17; no migration to any Vaultory table |
| **Testing** | Vitest, Playwright, and Go's existing suites as a regression guard |
| **Unknowns** | None. Eight research decisions, all resolved from Better Auth's source |

## Constitution Check

| Principle | Assessment |
|---|---|
| **I. Collector-First** | Recovery pages reuse `Field`, `Input`, `Button`, so both appearances work from one definition. Failures report every problem at once, against the field responsible, announced to assistive technology. The unverified banner informs rather than blocks — an unverified collector still has their vault |
| **II. Separation** | Unchanged from feature 004. Recovery is credentials and sessions, which live in Better Auth; Go still verifies every session itself and authorizes every request. No backend change at all, which is worth stating because the instinct on a security feature is to reach for the backend |
| **III. Contract-First** | The OpenAPI contract does not change, and a task verifies the generated types are byte-identical. Better Auth's recovery routes stay out of it: they do not cross the frontend/backend seam |
| **IV. Data Integrity and Security** | The whole of this feature. Tokens stored hashed (FR-015), never logged (FR-016), refusals indistinguishable (FR-017), requests non-revealing (FR-008), sessions revoked on reset (FR-018). Security-sensitive operations fail closed: anything unverifiable is refused with no content |
| **V. Quality, Simplicity, Spec-Driven** | Began from a specification with 28 requirements and 13 success criteria, clarified in three questions. Every walkthrough traces to requirements |

### Three defaults that had to be changed, and one that did not

Left alone, Better Auth would have quietly failed three requirements. Recorded here because each is
invisible when wrong:

- **Reset tokens are stored in plaintext by default.** A copy of `verification` would be a set of
  working reset links. `storeIdentifier: 'hashed'` fixes it.
- **`revokeSessionsOnPasswordReset` is off by default.** A reset would change the password and
  leave every session working — the key, not the lock.
- **Verification link lifetime defaults to 1 hour.** FR-003 allows 24, which is what somebody who
  registers at night needs.

The fourth was already right: `autoSignInAfterVerification` is opt-in, and FR-002a wants it off.

## Project Structure

```
specs/005-account-recovery/
├── spec.md              28 FRs, 13 SCs, 3 clarifications
├── plan.md              this file
├── research.md          8 decisions, from Better Auth's source
├── data-model.md        what changes in `verification`, and what does not
├── contracts/README.md  the two kinds of token, and why they differ
├── quickstart.md        walkthroughs A–I
└── checklists/requirements.md

frontend/
├── lib/auth.ts                          verification + reset config, hashed storage
├── lib/mail.ts                          NEW: send({to, subject, text})
├── app/(auth)/forgot-password/page.tsx  NEW
├── app/(auth)/reset-password/page.tsx   NEW
├── app/(auth)/verify-email/page.tsx     NEW: confirm, then offer sign-in
└── components/collection/VaultHeader.tsx  unverified banner + resend

compose.yaml                             NEW service: mailpit
backend/                                 unchanged
```

## Phases

| Phase | Contents |
|---|---|
| **0 — Mail** | Mailpit in Compose; `lib/mail.ts`; prove a message is captured before any flow depends on it |
| **1 — Verification (US1)** | Config, the verify page, the unverified banner and resend |
| **2 — Reset (US2)** | Config with hashed storage, forgot and reset pages |
| **3 — Eviction (US3)** | `revokeSessionsOnPasswordReset`, and the test that two browsers diverge |
| **4 — Hardening** | Rate limits, indistinguishable refusals, no tokens in logs, accessibility |

Phase 0 first because every later phase depends on a message actually arriving, and a flow that
silently sends nowhere looks identical to one that works until somebody checks their inbox.

## Risks

| Risk | Handling |
|---|---|
| **A default silently un-does a requirement** — three already would have | Each is asserted in `auth-config.test.ts`, the way feature 004 pinned its hasher. A configuration that nothing checks is one a later edit can weaken unnoticed |
| **A token reaches a log** — easy to do while debugging | Walkthrough G greps the logs; a test asserts recovery events are recorded without their contents |
| **Refusals drift apart** — four failure paths, easy for one to answer differently | One test compares status, body and timing across all four rather than four tests each checking one |
| **Mailpit hides a production problem** — it accepts anything | The interface is identical in both, so only the transport differs. Production configuration is exercised by walkthrough I's port check and nothing more; a real send is not verifiable here |
| **Rate limits shadow each other** | Feature 004's lesson: a path's own rule does not inherit the global ceiling. Each recovery path is stated explicitly |

## Complexity Tracking

| Deviation | Why needed | Simpler alternative rejected because |
|---|---|---|
| **FR-004 and FR-006 narrow to reset tokens** | Verification tokens are self-contained JWTs — nothing is stored, so nothing can be spent or invalidated. Neither requirement is achievable for them at any price the library allows | A denylist of spent tokens, or a per-user counter baked into the token, would buy single-use for verification at the cost of a table, a cleanup job, and a mechanism the library would fight on upgrade — to prevent an action whose effect is setting a boolean that is already true. The consequence of a replayed *reset* link is account takeover; of a replayed *verification* link, nothing |

**The spec is amended rather than left aspirational.** A requirement that cannot hold is worse than
one never written, because it reads as a guarantee to everyone after you.

### FR-012 was reconsidered separately, and is implemented rather than narrowed

The narrowing above covers verification tokens only. Cross-artifact analysis found that FR-012 — a
new reset request invalidates the previous link — also does not hold by default: `forget-password`
inserts a row and deletes nothing, so with FR-020's limit of three per hour, three working reset
links can exist at once.

It is **not** narrowed, because the two cases are not alike. A replayed verification link sets a
boolean that is already true; a stale reset link is a way into a vault. It is also cheap to fix:
`verification.value` holds the account id, so prior rows are findable and deletable even though
identifiers are hashed. T022a does it.

### FR-014 is implemented, and adds a session route on purpose

The same analysis found that completing a reset does not sign the collector in — Better Auth's
`password.mjs` has no `setSessionCookie` and no `createSession`, and no option adds one. Left
alone, the feature's headline journey ends with the collector signed out, looking at a sign-in form
immediately after proving their identity.

T021a signs them in with the password they have just chosen. That is a legitimate route precisely
because it produces an ordinary session the Go service verifies on the next request — it is not a
session minted by some other means, which FR-025 forbids. T034a exists to hold that line, since
this is the feature where it is most likely to slip.

No other deviation.

## Post-Design Constitution Re-check

Re-evaluated after the artifacts above: **no new violations**, and one entry in Complexity
Tracking.

The design improves on feature 004 in one respect worth noting. That feature shipped with a known
hole — a collector who forgot their password was locked out permanently — recorded honestly in its
Assumptions. This closes it, and the sign-in rate limit's deliberate refusal to ever permanently
lock an account stops being a mitigation for a missing feature and becomes ordinary prudence.
