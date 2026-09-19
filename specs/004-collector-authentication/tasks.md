# Tasks: Collector Authentication

**Input**: Design documents from `specs/004-collector-authentication/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/README.md, quickstart.md

**Tests**: Included. The constitution requires unit, integration, contract, and end-to-end suites
to pass before a feature is complete, and this feature's central requirement — that Go verifies
rather than trusts — is only meaningful if something adversarial exercises it.

**Organization**: By user story, after a foundation that all three depend on.

## Format: `[ID] [P?] [Story] Description`

- **[P]** — touches different files from every other [P] task in the same phase, with no dependency
  on incomplete work
- **[US1] / [US2] / [US3]** — the user story a task serves; Setup, Foundational and Polish carry none

## Path Conventions

`backend/` is the Go service, `frontend/` the Next.js app, both at the repository root.

---

## Phase 1: Setup

**Purpose**: Clear the ground. The first task exists because the feature cannot be installed today.

- [ ] T001 Upgrade `vitest` and `@vitest/*` to ^3 in `frontend/package.json`, then confirm all 33 existing tests still pass with `npm run test` — research Decision 9. Done first and alone: `better-auth` cannot be installed while `vitest@2` pins vite 5, and a failure here must be attributable to the upgrade rather than to authentication
- [ ] T002 Install `better-auth` in `frontend/package.json` with `npm install better-auth`, verifying it resolves without `--legacy-peer-deps`
- [ ] T003 [P] Add `BETTER_AUTH_SECRET`, `BETTER_AUTH_URL`, and `BETTER_AUTH_DATABASE_URL` to the `frontend` service in `compose.yaml` and `compose.prod.yaml`. The secret comes from the same variable the backend reads so the two cannot disagree (contracts/README.md). The database URL uses the **restricted role** created in T007a — never the one the backend uses
- [ ] T004 [P] Document the three new variables in `README.md`, including that `VAULTORY_DEV_IDENTITY` is being removed, and state plainly that the frontend now holds a database connection and why it is restricted

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The schema, and the Go verification that every user story rests on.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

### Schema

- [ ] T005 Generate the Better Auth schema with `npx @better-auth/cli generate` and transcribe it by hand into `backend/migrations/000007_create_auth_tables.up.sql` — `user`, `session`, `account`, `verification`, `rateLimit` per data-model.md. Better Auth must never run migrations against this database: one tool only, research Decision 4
- [ ] T006 Write `backend/migrations/000007_create_auth_tables.down.sql` dropping the five tables in reverse dependency order
- [ ] T007 Write `backend/migrations/000008_link_collectors_to_accounts.up.sql` in this order, which matters: delete the two seeded fixtures from migration 000005 and everything they own (FR-017a), add `collectors.user_id text`, backfill nothing because none can exist, then apply `UNIQUE NOT NULL REFERENCES "user"(id) ON DELETE CASCADE`. `NOT NULL` is only possible once no accountless collector remains
- [ ] T007a Create a dedicated PostgreSQL role for Better Auth in `backend/migrations/000008_link_collectors_to_accounts.up.sql`, with privileges on `user`, `session`, `account`, `verification` and `rateLimit` **only**. Explicitly no `SELECT` on `collectors`, `collectibles`, `collectible_images`, or `collectible_submissions`. This is what keeps the plan's claim true — that a frontend bug cannot expose another collector's vault — once Next.js holds a database connection. Revoke in the down migration
- [ ] T008 Add the `AFTER INSERT ON "user"` trigger to `000008_link_collectors_to_accounts.up.sql`, inserting a `collectors` row with a fresh uuid — research Decision 3. This is the only way a collector comes into existence
- [ ] T008a Make the trigger function `SECURITY DEFINER` and owned by the migration role, so it can insert into `collectors` while the Better Auth role that fires it cannot. The privilege belongs to the trigger, not to the caller. Set an explicit `search_path` on the function — a `SECURITY DEFINER` function without one is a privilege-escalation vector
- [ ] T009 Write `backend/migrations/000008_link_collectors_to_accounts.down.sql` dropping the trigger and column and re-seeding the two fixtures, so the migration is reversible as Principle IV requires
- [ ] T010 Apply both migrations against the test database and confirm they run clean up, down, and up again

### Go verification — the security boundary

- [ ] T011 Write `backend/internal/identity/session.go`: URL-decode the cookie, split on the **last** `.`, recompute `HMAC-SHA256(secret, token)` and compare in constant time. Use `base64.StdEncoding` — Better Auth signs with `btoa`, so this is standard base64, not the `RawURLEncoding` feature 001's dev cookie used. Getting this wrong fails every verification (contracts/README.md)
- [ ] T012 Add the resolver query to `backend/internal/identity/session.go`: one statement joining `session → user → collectors`, requiring `expiresAt > now()` and `createdAt > now() - interval '90 days'`, returning `collectors.id`. The 90-day cap is enforced here and nowhere else — research Decision 6
- [ ] T013 [P] Unit-test the signature verification in `backend/tests/unit/session_cookie_test.go` against a fixture **generated by Better Auth itself**, not by our own Go code. A fixture we generate would prove only that our signer matches our verifier
- [ ] T014 Wire the new resolver in `backend/cmd/vaultory-api/main.go`, keeping the refusal to start when no resolver is configured and removing the development branch
- [ ] T015 Delete `backend/internal/identity/dev.go` and `backend/internal/identity/dev_production.go`, and remove `/api/dev/session` from `backend/internal/transport/httpapi/server.go` — research Decision 8
- [ ] T016 Remove `cfg.DevIdentity` from `backend/internal/config/config.go` and `VAULTORY_DEV_IDENTITY` from `compose.yaml`
- [ ] T017 Replace the now-vacuous `go test -tags production` step in `compose.yaml` with one that still means something, and drop the `production` build tag from `backend/Dockerfile` if nothing else needs it — research Decision 8. A test asserting a deleted file is absent passes trivially
- [ ] T018 Update `backend/tests/contract/helper_test.go` and `backend/tests/integration/main_test.go` to create sessions by inserting rows directly, since `identity.NewDevResolver` no longer exists

### Adversarial tests — run before any UI exists

- [ ] T019 [P] Integration test in `backend/tests/integration/session_verification_test.go`: absent, expired, past-cap, tampered-signature, unsigned-token, and unknown-token sessions are each refused (FR-014, SC-004)
- [ ] T020 [P] Contract test in `backend/tests/contract/asserted_identity_test.go`: a collector id supplied in a header, a query parameter, and a body field is ignored, with and without a valid session for someone else (FR-013, SC-005). This is the test that would catch the mistake the whole architecture exists to prevent
- [ ] T020a [P] Integration test in `backend/tests/integration/auth_role_privileges_test.go`: connecting as the Better Auth role, a `SELECT` against `collectors`, `collectibles`, `collectible_images` and `collectible_submissions` is **refused by PostgreSQL**, while its own five tables are readable and registration still creates a collector through the trigger. Grants asserted against the database, not assumed from the migration text
- [ ] T021 [P] Integration test in `backend/tests/integration/session_cap_test.go`: a session aged past 90 days with a healthy `expiresAt` is refused — the case Better Auth alone would let through

**Checkpoint**: Go verifies sessions correctly and provably, with no frontend involved. User story work can begin.

---

## Phase 3: User Story 1 — Create an account and start a vault (Priority: P1) 🎯 MVP

**Goal**: A visitor registers and lands in their own empty, private vault.

**Independent Test**: Register a new account, confirm the vault is empty and accepts a collectible, with no development endpoint available anywhere.

- [ ] T022 [US1] Create the Better Auth server instance in `frontend/lib/auth.ts`: email and password enabled, PostgreSQL reached through `BETTER_AUTH_DATABASE_URL` using the restricted role from T007a, `emailAndPassword.minPasswordLength` 12 (FR-004), no social providers
- [ ] T023 [US1] Mount Better Auth's routes at `frontend/app/api/auth/[...all]/route.ts`
- [ ] T024 [P] [US1] Create the client helpers in `frontend/lib/auth-client.ts`
- [ ] T025 [US1] Build the registration page at `frontend/app/(auth)/register/page.tsx`, reusing `Field`, `Input` and `Button` so it looks like the rest of Vaultory and inherits both appearances
- [ ] T026 [US1] Report registration failures the way the add form does (FR-026): every problem at once, against the field responsible, `role="alert"`, and never discarding what was typed
- [ ] T027 [P] [US1] Lower-case the email before storing and comparing so uniqueness and sign-in are case-insensitive (FR-003)
- [ ] T028 [US1] Point the landing page's call to action at `/register` in `frontend/app/page.tsx`, replacing the temporary `ENTER_VAULT` constant (FR-024)
- [ ] T029 [P] [US1] Unit tests in `frontend/tests/unit/register-form.test.tsx`: short password, duplicate email, and mixed-case duplicate are each refused with a field-level message
- [ ] T030 [P] [US1] E2E test in `frontend/tests/e2e/register.spec.ts` following quickstart walkthrough A, including that exactly one `user` and one `collectors` row exist afterwards — the trigger, observed rather than assumed

**Checkpoint**: A person can create an account and own a vault. This is the MVP.

---

## Phase 4: User Story 2 — Sign in and stay signed in (Priority: P1)

**Goal**: A returning collector signs in and is not asked again until the session expires.

**Independent Test**: Sign in, close the browser, reopen it, reach the collection without re-entering credentials.

- [ ] T031 [US2] Build the sign-in page at `frontend/app/(auth)/sign-in/page.tsx`, linking to registration and back (FR-021)
- [ ] T032 [US2] Give an unknown email and a wrong password the same refusal, with no timing difference worth measuring (FR-008, SC-006)
- [ ] T033 [US2] Configure session lifetime in `frontend/lib/auth.ts`: `expiresIn` 30 days with `updateAge` so expiry is extended on use — the sliding half of FR-010; the cap is already enforced in Go by T012
- [ ] T034 [P] [US2] Configure Better Auth's rate limiting for sign-in: 10 failures per account per 15 minutes, then 15 minutes of refusal that lifts by itself, with a message saying when to retry (FR-027). Never a permanent lock — password reset is out of scope, so a locked-out collector would have no way back in
- [ ] T035 [P] [US2] Show who is signed in, and a sign-out control, on every vault page (FR-025)
- [ ] T036 [P] [US2] Contract test in `backend/tests/contract/signin_response_test.go` comparing status, body, and timing for an unknown email against a wrong password (SC-006). Not a walkthrough: eyeballing two responses is not evidence
- [ ] T037 [P] [US2] E2E test in `frontend/tests/e2e/sign-in.spec.ts` following quickstart walkthrough B, including that a session survives a browser restart

**Checkpoint**: Registration and sign-in both work, independently.

---

## Phase 5: User Story 3 — Sign out, and be told when you are not signed in (Priority: P2)

**Goal**: Signing out ends the session immediately, and a signed-out visitor is invited in rather than shown a broken service.

**Independent Test**: Sign out and confirm the captured cookie no longer reaches any collection content; open a vault page signed out and confirm you arrive at sign-in.

- [ ] T038 [US3] Implement sign-out, deleting the session row so the next request finds nothing (FR-011). Immediate revocation is why research Decision 1 chose a database lookup over a self-contained token
- [ ] T039 [US3] Redirect a signed-out request for a vault page to `/sign-in?next=<path>` and return the collector there after signing in (FR-022), replacing the error boundary's "your collection could not be loaded" for the 401 case
- [ ] T040 [US3] Write `frontend/lib/safe-redirect.ts` rejecting any `next` that is not a path within Vaultory — absolute URLs, protocol-relative `//host`, and anything with a scheme (FR-022a). A destination taken from a request and followed after sign-in is an open redirect
- [ ] T041 [P] [US3] Unit tests in `frontend/tests/unit/safe-redirect.test.ts` covering `https://example.com`, `//example.com`, `/\\example.com`, `javascript:`, and the legitimate `/collection`
- [ ] T042 [P] [US3] Ensure no vault page is served from the browser's back-forward cache after sign-out (US3 scenario 3)
- [ ] T043 [P] [US3] E2E test in `frontend/tests/e2e/sign-out.spec.ts` following quickstart walkthrough E, including the open-redirect cases

**Checkpoint**: All three user stories work independently.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [ ] T044 Confirm the production image starts and serves traffic (FR-019, SC-009). It has refused since feature 002, correctly, for want of a resolver — quickstart walkthrough G
- [ ] T045 Inspect the shipped binary for any trace of the development resolver (SC-008), the way feature 002 did. A passing test proves a file was deleted; the artifact proves what ships
- [ ] T046 [P] Verify `frontend/lib/types/api.ts` is byte-identical to before this feature — the OpenAPI contract does not change, and contracts/README.md says so rather than assuming it
- [ ] T047 [P] Log registration, sign-in, sign-in failure, and sign-out without credentials (FR-028)
- [ ] T048 [P] Confirm both auth pages are keyboard-operable and announce failures to assistive technology (SC-011)
- [ ] T049 [P] Remove the development sign-in from `README.md` and `specs/001-add-browse-collectibles/quickstart.md`, which still instruct people to use it (FR-018)
- [ ] T050 Walk quickstart walkthroughs A–I and record the result of each, marking anything not observed as unverified rather than assumed
- [ ] T051 Run the full suite — `make test` and `make test-e2e` — and record the outcome per browser project. Desktop passes today; tablet and mobile sit at roughly 17/44 from a defect predating this feature, so a fair comparison needs the before-figure stated

---

## Dependencies

```
Phase 1 Setup  ──>  Phase 2 Foundational  ──┬──>  Phase 3 US1 ──┐
                                            ├──>  Phase 4 US2 ──┼──>  Phase 6 Polish
                                            └──>  Phase 5 US3 ──┘
```

- **T001 blocks T002**, which blocks every frontend task. Nothing installs until vitest moves.
- **T005–T010 (including T007a, T008a) block T011–T021**: there is nothing to query until the tables exist.
- **T011–T021 block all three user stories**, and are deliberately testable without any UI — insert
  a session row and make a request.
- **US1 and US2 both depend on T022** (the Better Auth instance). US3 depends on US2 in practice:
  signing out requires signing in first.
- T039 supersedes the 401 error state, so it must land after the pages it redirects to exist.

## Parallel Opportunities

- **Setup**: T003 and T004 together, after T002.
- **Foundational**: T013, T019, T020, T020a and T021 are separate test files and run together once T012
  lands. T005–T010 are strictly sequential — migrations are ordered by nature.
- **US1**: T024, T027 and T029 alongside the page work.
- **US2**: T034, T035, T036 and T037 together.
- **US3**: T041, T042 and T043 together, after T040.
- **Polish**: T046, T047, T048 and T049 together.

## Implementation Strategy

**MVP is Phase 1 + Phase 2 + Phase 3.** That delivers a person who can create an account and own a
private vault, on a backend that verifies every session itself.

Phase 2 before any UI is the deliberate part. The security boundary gets adversarial tests — T019,
T020, T021 — while there is no interface to make it *look* like it works. A forged header ignored
by a Go handler is a fact; a login page that appears to work proves nothing about what the server
accepts.

**Two things are worth doing early and separately.** T001, because the feature cannot be installed
until it lands and a failure there has nothing to do with authentication. And T017, because
deleting the dev resolver quietly turns an existing production-build assertion into one that passes
for the wrong reason.

**Expect T005 to be tedious and worth the tedium.** Transcribing a generated schema by hand is
dull, and it is what keeps one migration tool in charge of one database.
