# Tasks: Lock Sign-In per Email Address After Repeated Failures

**Input**: `specs/013-sign-in-lockout/` — spec, plan, research, data-model, contracts/sign-in.md

## Phase 1: Setup

- [X] T001 Create `backend/migrations/000013_create_sign_in_lockouts.up.sql`: table `sign_in_lockouts` (data-model.md), functions `sign_in_is_locked`, `sign_in_record_failure`, `sign_in_clear`, and grants to `vaultory_auth` on exactly the table and the three functions; and the `.down.sql` that drops them

## Phase 2: Foundational

- [X] T002 [P] `backend/tests/integration/auth_role_privileges_test.go`: the auth role may use `sign_in_lockouts` and execute the three functions, and still cannot read any collection table
- [X] T003 `frontend/lib/sign-in-lockout.ts`: `addressKey(email, secret)` — trim, lower-case, HMAC-SHA256, base64url; `isLocked`, `recordFailure`, `clear` calling the functions; `LOCKED_MESSAGE`, `MAX_FAILURES = 5`, `LOCK_MINUTES = 15` (research Decisions 2, 5)

## Phase 3: User Story 1 — Guessing stops after five tries (P1) 🎯 MVP

**Independent test**: five failures for one address, the sixth refused even with the right password; lifts after 15 minutes.

- [X] T004 [US1] `backend/tests/integration/sign_in_lockout_test.go`: five failures lock; a locked address stays locked and its count does not move; four failures then a clear start again from zero; an expired lock is cleared by the next failure, which counts as one; 20 concurrent failures are all counted (SC-005)
- [X] T005 [US1] `frontend/lib/auth.ts`: `hooks.before` refuses a locked address on `/sign-in/email` with 429 `SIGN_IN_LOCKED` and `LOCKED_MESSAGE`; `hooks.after` records a 401 `INVALID_EMAIL_OR_PASSWORD` as a failure and clears on success; nothing for 403 `EMAIL_NOT_VERIFIED` or 429 (FR-002–FR-004, FR-010)
- [X] T006 [P] [US1] `frontend/tests/unit/sign-in-lockout.test.ts`: the key is the same for case and whitespace variants, differs between addresses and secrets, and never contains the address; the hook wiring calls record on 401, clear on success, nothing on 403; a locked address throws 429 with the message

## Phase 4: User Story 2 — Nobody learns which addresses have accounts (P1)

**Independent test**: identical responses for a registered and an unregistered address across six failures.

- [X] T007 [US2] `frontend/tests/e2e/sign-in-lockout.spec.ts`: for a registered and an unregistered address, six wrong attempts give the same sequence of status and body — five 401s, then 429 `SIGN_IN_LOCKED`; the registered account's right password is refused while locked (FR-006, SC-001, SC-002)
- [X] T008 [P] [US2] `frontend/tests/e2e/signin-indistinguishable.spec.ts`: correct the comment that says the limit is 10 per account (it stays under 5 at four rounds)

## Phase 5: User Story 3 — A locked-out collector can get back in (P2)

- [X] T009 [US3] `frontend/lib/auth.ts`: `emailAndPassword.onPasswordReset` clears the account's address (FR-005)
- [X] T010 [US3] Same e2e file as T007: lock an account, complete a reset through Mailpit, sign in with the new password at once; requesting a reset alone leaves the lock (SC-003)

## Phase 6: Polish

- [X] T011 `frontend/lib/auth.ts`: correct the comment beside the sign-in limit — it is the per-network ceiling, not FR-027, and accounts are now locked because reset exists (FR-011, FR-012)
- [X] T012 [P] `specs/004-collector-authentication/spec.md`: amend FR-027 to five consecutive failures per address with the generic message, and the clarification that chose ten (FR-012)
- [X] T013 Full gate: backend gofmt, vet (± integration tag), build, unit, integration, contract; frontend lint, typecheck, unit, build. Playwright needs Docker: written here, run by the browser CI job

## Dependencies

T001 → T002, T004; T003 → T005, T006, T009; T005 → T007; T009 → T010.

## As run

**T013** (2026-11-20, Go 1.26.10, PostgreSQL 17.6, Node 22): migration 000013 applies, reverses and re-applies; backend gofmt, vet (± integration tag), build, unit, integration and contract pass, and the lockout tests also under `-race`; frontend lint, typecheck, unit (131) and build pass.

**The hooks against the real server**, because the unit tests use a stand-in for the database: the built app on PostgreSQL with a local SMTP sink, driven with curl. A registered and an unregistered address got identical status and body on all six attempts — five 401s, then 429 `SIGN_IN_LOCKED`; the right password, and the address in upper case with spaces, were refused the same way while locked; requesting a reset left the lock; completing it let the new password sign in at once; four failures, a success, four more failures did not lock; no stored key contains an address.

**T007 and T010** are written and type-checked but not run here: the browser suite needs Docker, and runs in the browser CI job.

