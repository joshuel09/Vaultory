# Tasks: Account Recovery

**Input**: Design documents from `specs/005-account-recovery/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/README.md, quickstart.md

**Tests**: Included. The constitution requires the suites to pass before a feature is complete, and
this feature's requirements are almost all invisible when they fail — a token stored in plaintext,
a session that survives a reset, and a refusal that reveals too much all look exactly like working
software.

**Organization**: By user story, after mail works. Nothing else can be checked until a message
actually arrives.

## Format: `[ID] [P?] [Story?] Description`

- **[P]** — different files from every other [P] task in the same phase, no dependency on
  incomplete work
- **[US1] / [US2] / [US3]** — the user story served; Setup, Foundational and Polish carry none

## Path Conventions

`frontend/` is the Next.js app, `backend/` the Go service (**unchanged in this feature**), both at
the repository root.

---

## Phase 1: Setup

- [ ] T001 Add a `mailpit` service to `compose.yaml` on the dev profile: `axllent/mailpit`, SMTP on 1025 and the web interface on `${MAILPIT_PORT:-8025}`, in-memory storage. It holds messages and has no outbound path, so a message cannot escape by misconfiguration — that is what makes it safe to point a real send at locally (research Decision 6)
- [ ] T002 Install `nodemailer` and `@types/nodemailer` in `frontend/package.json`. Remember feature 004's lesson: the container mounts `node_modules` as a named volume that shadows the image, so a new dependency needs an install inside the running container or a fresh volume
- [ ] T003 [P] Add `MAIL_HOST`, `MAIL_PORT`, `MAIL_FROM` to the `frontend` service in `compose.yaml` and `compose.prod.yaml`. Production additionally needs `MAIL_USER` and `MAIL_PASSWORD`, both required with `${VAR:?}` — and check the full set is supplied to `make prod-build` and `make check`, which have now broken twice for exactly this reason
- [ ] T004 [P] Document the new variables and Mailpit in `README.md`, including that development mail never leaves the machine

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Make a message arrive. Everything after this depends on it.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T005 Write `frontend/lib/mail.ts` exposing exactly `send({ to, subject, text })`, backed by nodemailer. One interface so the transport is swappable and no call site learns which one is in use (contracts/README.md)
- [ ] T006 Wire `lib/mail.ts` to Mailpit in development (FR-023) and to SMTP credentials in production (FR-022), from the variables added in T003
- [ ] T007 [P] Unit test in `frontend/tests/unit/mail.test.ts`: `send` passes recipient, subject and body through, and surfaces a transport failure rather than swallowing it. A send that fails silently is indistinguishable from one that worked until somebody checks their inbox
- [ ] T008 Verify by hand that a message sent from the running stack appears at <http://localhost:8025>, and that Mailpit publishes no outbound path (FR-023, SC-008), before building any flow on top of it

**Checkpoint**: mail works and is inspectable. User story work can begin.

---

## Phase 3: User Story 1 — Prove the address is mine (Priority: P1) 🎯 MVP

**Goal**: A collector registers, receives a message, follows the link, and their address is verified.

**Independent Test**: Register, follow the link from Mailpit, and confirm the account moves from unverified to verified — and that an unfollowed link leaves it unverified.

- [ ] T009 [US1] Enable verification in `frontend/lib/auth.ts`: `emailVerification.sendOnSignUp` (FR-001), a `sendVerificationEmail` that calls `lib/mail.ts`, and `expiresIn` of 24 hours. The default is 1 hour; FR-003 allows a day, which is what somebody who registers at night and reads their mail next morning needs (research Decision 5)
- [ ] T010 [US1] Leave `autoSignInAfterVerification` unset, and assert that in `frontend/tests/unit/auth-config.test.ts` (FR-002a). It is opt-in, so this is a test guarding an absence — the convenience of signing someone in is exactly the edit a later contributor would make, and it would turn a 24-hour email into a way into a vault
- [ ] T011 [US1] Build `frontend/app/(auth)/verify-email/page.tsx`: confirm the address is verified, then send the collector to their collection if they have a session and to sign-in if they do not (FR-002, FR-002a)
- [ ] T012 [P] [US1] Show an unverified collector that their address is unverified, with a resend control, in `frontend/components/collection/VaultHeader.tsx` (FR-005). It informs; it must not block — an unverified collector still has their vault
- [ ] T013 [US1] Implement resend. A previous verification link stays valid until it expires, because verification tokens are stateless JWTs and there is nothing stored to invalidate — record that in a comment so the next reader knows it was decided rather than missed (FR-006 as narrowed; research Decision 3). Note the contrast with reset links, which T022a *does* invalidate, because those grant access
- [ ] T014 [P] [US1] Compose the verification message in `frontend/lib/messages.ts`: what it is for, who it is for, how long the link lasts, and what to do if they did not ask for it (FR-024)
- [ ] T015 [P] [US1] E2E test in `frontend/tests/e2e/verify-email.spec.ts` following quickstart walkthroughs A and B (SC-002, SC-013), reading the link from Mailpit's API. **Walkthrough B is the one that matters**: following the link with no session must verify the address and grant nothing

**Checkpoint**: an address can be proven. Reset now has a channel it can trust.

---

## Phase 4: User Story 2 — Get back into my vault (Priority: P1)

**Goal**: A collector who cannot remember their password sets a new one and reaches their collection.

**Independent Test**: Request a reset, follow the link, set a new password, sign in with it — and confirm the old one is refused.

- [ ] T016 [US2] Enable reset in `frontend/lib/auth.ts`: `emailAndPassword.sendResetPassword` calling `lib/mail.ts`, and `resetPasswordTokenExpiresIn` left at its 1-hour default, which is already FR-010
- [ ] T017 [US2] Set `verification.storeIdentifier: 'hashed'` in `frontend/lib/auth.ts` (FR-015). **This is the single most consequential line in the feature.** By default Better Auth stores the token itself, so a copy of the `verification` table would be a set of working reset links to every account with an outstanding reset — and nothing about the running system would look wrong
- [ ] T018 [P] [US2] Assert both of the above in `frontend/tests/unit/auth-config.test.ts`: storage is hashed and not plain, and the reset lifetime is an hour. A configuration that nothing checks is one a later edit can weaken unnoticed — the same reasoning that pinned the password hasher in feature 004
- [ ] T019 [US2] Build `frontend/app/(auth)/forgot-password/page.tsx`: ask for an address, send a link only when it has an account (FR-009), and answer identically either way (FR-007, FR-008)
- [ ] T020 [US2] Build `frontend/app/(auth)/reset-password/page.tsx`: accept the token, require a password meeting the same minimum as registration, and refuse without consuming the link (FR-013)
- [ ] T021 [P] [US2] Link "forgotten your password" from `frontend/app/(auth)/sign-in/page.tsx`
- [ ] T021a [US2] Sign the collector in once a reset completes (FR-014). Better Auth does **not** do this: `password.mjs` contains no `setSessionCookie` and no `createSession`, and no option changes it — the endpoint returns `{status:true}` and leaves them staring at a sign-in form immediately after proving their identity. Sign in with the password they just chose, which they demonstrably know because they typed it. This must run **after** `revokeSessionsOnPasswordReset` (T026), or the new session is deleted along with the old ones
- [ ] T021b [P] [US2] Assert in `frontend/tests/e2e/reset-password.spec.ts` that the collector lands in their collection after a reset rather than on a sign-in form, and that the session they arrive with is accepted by **Go on :8080** (FR-014, FR-025). The second half is the point: signing in after a reset is legitimate only because it produces an ordinary session the backend verifies, and a shortcut that skipped that would look identical from the browser
- [ ] T022 [US2] Mark the address verified when a reset completes, if it was not already (FR-014a, SC-012). Following a link sent to that address proves control of the inbox, which is what verification tests
- [ ] T022a [US2] Invalidate any outstanding reset link when a new one is requested (FR-012). Better Auth does not: `forget-password` calls `createVerificationValue`, which inserts a row and deletes nothing, so with the FR-020 limit of three per hour **three working reset links can exist at once**, each for an hour. Delete prior `reset-password:` rows for that account first — `verification.value` holds the account id, so they are findable even though identifiers are hashed
- [ ] T022b [P] [US2] Test in `frontend/tests/e2e/reset-password.spec.ts` that requesting a second reset makes the first link stop working (FR-012). Research Decision 3 narrowed single-use for *verification* tokens, where a replay does nothing; this is the other case, where a stale link is a way into a vault
- [ ] T023 [P] [US2] Compose the reset message in `frontend/lib/messages.ts`, stating the one-hour lifetime and what to do if they did not request it (FR-024)
- [ ] T024 [P] [US2] E2E test in `frontend/tests/e2e/reset-password.spec.ts` following quickstart walkthrough C, including the four refusals: short password, second use (FR-011), expired, altered. Also that the whole journey completes in a couple of minutes (SC-001)
- [ ] T025 [P] [US2] Integration test in `frontend/tests/e2e/reset-token-storage.spec.ts` following walkthrough F: request a reset, read `verification.identifier` from the database, confirm it is **not** the token from the link, and confirm the stored value **does not work** as a token (SC-009). Asserting the configuration is not the same as proving the property

**Checkpoint**: a forgotten password is recoverable. The hole feature 004 shipped with is closed.

---

## Phase 5: User Story 3 — A reset evicts whoever prompted it (Priority: P2)

**Goal**: Resetting a password ends every other session immediately.

**Independent Test**: Sign in on two browsers, reset from one, confirm the other can no longer reach the collection.

- [ ] T026 [US3] Set `emailAndPassword.revokeSessionsOnPasswordReset: true` in `frontend/lib/auth.ts` (FR-018). It is **off by default**: left alone, a reset changes the password and leaves every session working, which changes the key without changing the lock
- [ ] T027 [P] [US3] Assert it in `frontend/tests/unit/auth-config.test.ts`. Another test guarding a default that would silently undo a requirement
- [ ] T028 [US3] E2E test in `frontend/tests/e2e/reset-evicts.spec.ts` following walkthrough D: two browser contexts, reset from one, and the other's captured cookie is refused **by Go on :8080** (SC-005). Checking it against the backend directly is what proves the eviction is real rather than a cleared cookie
- [ ] T029 [P] [US3] Confirm the old password is refused after a reset (FR-019, SC-006)

**Checkpoint**: all three user stories work independently.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [ ] T030 Configure rate limits in `frontend/lib/auth.ts` for the recovery paths: three per hour each, verification and reset counted separately (FR-020). Feature 004's lesson applies — a path's own rule does not inherit the global ceiling, and Better Auth applies stricter built-in defaults to some paths, so each is stated explicitly
- [ ] T031 [P] E2E test following walkthrough H: the fourth verification request in an hour is refused **while a reset request for the same address still succeeds** (SC-011). Exhausting one limit must not block the other at the moment it is most needed
- [ ] T032 Test in `frontend/tests/e2e/recovery-refusals.spec.ts` that expired, spent, altered and foreign tokens are refused **indistinguishably** — one test comparing status, body and timing across all four, rather than four tests each checking one (FR-017, SC-004)
- [ ] T033 [P] Test that a reset request for an address with an account and one without are indistinguishable in status, body and timing (FR-008, SC-003)
- [ ] T034 [P] Record recovery events — requested, sent, completed, refused — in `frontend/app/api/auth/[...all]/route.ts`, without tokens (FR-021). The existing handler already logs action and status without reading the body; confirm the recovery paths are covered
- [ ] T034a Test that recovery introduces no new route to a session (FR-025, FR-026). After registering, verifying, requesting a reset and completing one, every session that exists must be a row the Go resolver accepts, and a request carrying an asserted identity instead of a session must still be refused. This feature adds two ways to become authenticated, which is when feature 004's guarantee is most likely to be undone by accident — and T021a deliberately adds one of them
- [ ] T035 Verify no token reaches a log, following walkthrough G: grep the running stack's output for token-shaped strings and full links after exercising every flow (FR-016, SC-007). Inspect the output rather than assert the intention
- [ ] T036 [P] Confirm recovery pages are keyboard-operable and announce failures to assistive technology, extending `frontend/tests/e2e/auth-accessibility.spec.ts` (SC-010)
- [ ] T037 [P] Verify `frontend/lib/types/api.ts` is byte-identical — the OpenAPI contract does not change, and contracts/README.md says so rather than assuming it
- [ ] T038 Amend `spec.md` to narrow FR-004 and FR-006 to reset tokens, with the reason. Verification tokens are stateless JWTs and neither requirement can hold for them; leaving them as written would read as a guarantee (plan.md Complexity Tracking)
- [ ] T039 Walk quickstart walkthroughs A–I and record the result of each, marking anything not observed as unverified rather than assumed
- [ ] T040 Run the full suites — `make test`, and the browser suite per project — and record the outcome. Desktop, tablet and mobile were all green at the end of #22; a regression here is attributable

---

## Dependencies

```
Phase 1 Setup ──> Phase 2 Mail ──┬──> Phase 3 US1 ──> Phase 4 US2 ──> Phase 5 US3 ──> Phase 6
                                 │         (verification gives reset a channel to trust)
                                 └──> T007 tests can run alongside
```

- **T001–T002 block T005**: there is nothing to send through until Mailpit and nodemailer exist.
- **Phase 2 blocks everything.** A flow that silently sends nowhere looks exactly like one that
  works, until somebody checks their inbox.
- **US2 depends on US1 in substance, not just sequence.** A reset link may only be sent to an
  address somebody has proven they control; that is why these are one feature.
- **US3 depends on US2**: there is no reset to evict sessions from until reset exists.
- **T021a depends on T026.** Signing in after a reset must happen after session revocation, or the
  new session is revoked with the old ones. Ordered across phases deliberately.
- T017 and T026 are one line each and are the two most consequential lines in the feature.

## Parallel Opportunities

- **Setup**: T003 and T004 together, after T002.
- **Foundational**: T007 alongside T005–T006.
- **US1**: T012 and T014 alongside the page work; T015 after.
- **US2**: T018, T021, T023 together; T021b, T022b, T024 and T025 after the pages exist.
- **US3**: T027 and T029 together.
- **Polish**: T031, T033, T034, T034a, T036 and T037 together.

## Implementation Strategy

**MVP is Phase 1 + Phase 2 + Phase 3.** That delivers a collector who can prove their address —
which is what makes everything after it safe rather than merely possible.

**Mail before flows, deliberately.** Every later phase depends on a message arriving, and the
failure mode is quiet: a send that goes nowhere produces a page that looks correct, an email that
never comes, and a bug report a week later.

**Three tasks are one line each and carry most of the risk**: T017 (hashed storage), T026 (session
revocation), T010 (no auto sign-in). Each changes or preserves a default that would otherwise
silently fail a requirement, and each is paired with a test that guards it — because a
configuration nothing checks is one the next edit can undo without anyone noticing.

**T038 is not paperwork.** Two requirements cannot hold as written, and a specification that
promises what the software cannot do misleads everyone after you. Amending it is part of finishing
the feature, not an afterthought.
