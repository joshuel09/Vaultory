# Tasks: An Audit Log of Account and Collection Changes

## Phase 1: Setup

- [ ] T001 `backend/migrations/000014_create_audit_events.{up,down}.sql`: table, index, the three triggers, `audit_purge_expired`, `INSERT` for `vaultory_auth` (data-model.md)

## Phase 2: Foundational

- [ ] T002 [P] `backend/tests/integration/auth_role_privileges_test.go`: the auth role may insert into `audit_events` and may not read, change or delete it (FR-008)

## Phase 3: User Story 2 — Collection changes (P1)

- [ ] T003 [US2] `backend/internal/store/postgres/audit.go`: `recordCollectibleEvent(ctx, tx, action, collectorID, collectibleID)` resolving the account from `collectors`; `PurgeExpiredAudit(ctx, limit)`
- [ ] T004 [US2] `backend/internal/store/postgres/collectibles.go`: `Add`, `Edit`, `Delete` record their event in their transaction (FR-001, FR-004)
- [ ] T005 [US2] `backend/tests/integration/audit_test.go`: create, edit, delete each record one entry with the account and collectible and nothing else; a refused edit (version conflict) and a delete of someone else's collectible record nothing; with the audit insert made to fail, the change is not applied (SC-004)

## Phase 4: User Story 3 — The record cannot be rewritten (P2)

- [ ] T006 [US3] Same test file: UPDATE, DELETE of a recent entry and TRUNCATE are refused as the owning role; a year-old entry is removed by `audit_purge_expired` and a recent one is not (FR-006, FR-007, SC-003)
- [ ] T007 [US3] `backend/internal/collection` and `cmd/vaultory-api/main.go`: purge at start-up and after each deletion (FR-007)

## Phase 5: User Story 1 — Account events (P1)

- [ ] T008 [US1] `frontend/lib/audit.ts`: `recordAccountEvent(db, event)` — inserts, and on failure logs a structured line with the action and never throws (FR-005)
- [ ] T009 [US1] `frontend/lib/auth.ts`: registration, sign-in succeeded / failed / refused, sign-out (path `/sign-out` only), email verified, password reset — research Decision 4
- [ ] T010 [P] [US1] `frontend/tests/unit/audit.test.ts`: each hook records its action and outcome; failures carry the address key, not the address; a failing insert does not throw
- [ ] T011 [US1] Drive the real server (built app, PostgreSQL, SMTP sink) through each account event and read the table: one entry each, nothing personal (SC-001, SC-002)

## Phase 6: Polish

- [ ] T012 Full gate: backend gofmt, vet (± integration tag), build, unit, integration, contract; frontend lint, typecheck, unit, build
