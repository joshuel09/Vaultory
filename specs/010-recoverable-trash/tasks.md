# Tasks: Undo and a Recoverable Trash

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/trash.md](contracts/trash.md), [quickstart.md](quickstart.md)

**Tests**: required — the constitution asks for important business rules to be tested, and SC-003
(nothing trashed leaks into a vault view) can only be proven by exercising every view.

## Phase 1: Setup

- [ ] T001 Write `backend/migrations/000012_add_collectible_trash.up.sql`: add `deleted_at timestamptz`
  and the two partial indexes; and `.down.sql` that queues trashed collectibles' image files, deletes
  those image rows and the trashed collectibles, then drops the indexes and column (data-model.md)
- [ ] T002 Add the three operations, `TrashedCollectible` and `TrashPage` to `backend/api/openapi.yaml`,
  and update `deleteCollectible`'s description to say it moves to the Trash (contracts/trash.md)
- [ ] T003 Regenerate `frontend/lib/types/api.ts` with `npm run generate:api`

## Phase 2: Foundational — the vault stops seeing the Trash

- [ ] T004 Add `AND c.deleted_at IS NULL` (or the unaliased form) to `Get`, the `Edit` lock and its
  read-back, `List` and its count, and `findBySubmissionKey` in
  `backend/internal/store/postgres/collectibles.go`, and to both queries in `statistics.go` (FR-002)
- [ ] T005 Replace `Store.Delete` with `Store.Trash` in `collectibles.go`: set `deleted_at = now()` only
  where it is NULL, delete the collectible's `collectible_submissions` row in the same transaction,
  report whether it moved (FR-001, FR-004, research Decision 5)

## Phase 3: User Story 1 — Undo a deletion immediately (P1) 🎯 MVP

- [ ] T006 [US1] Add `Store.Restore(collector, id, retention)` in `collectibles.go`: clear `deleted_at`
  where set and within the window, return the row; `ErrNotFound` otherwise (FR-007, FR-008)
- [ ] T007 [US1] In `backend/internal/collection/`: `trashRetention` constant; `Delete` trashes, logs
  "collectible moved to trash" with identifiers only, then purges and drains; `Restore` logs
  "collectible restored" (FR-001, FR-009, FR-015)
- [ ] T008 [US1] `POST /api/trash/{collectibleId}/restore` in
  `backend/internal/transport/httpapi/trash.go` and `server.go`: 200 with the collectible, 404 for a
  malformed or absent id (FR-007, FR-008)
- [ ] T009 [P] [US1] `frontend/lib/api/trash.ts`: `restoreCollectible`, `listTrash`, `deletePermanently`
- [ ] T010 [US1] Deleting lands on `/collection?deleted=<id>`
  (`EditCollectibleForm.tsx`); `UndoDeleteNotice.tsx` on the gallery offers Undo (restore, then
  refresh without the parameter) and a link to the Trash; shown once (FR-005)
- [ ] T011 [US1] `DeleteCollectibleDialog.tsx` copy: "Move … to the Trash?", kept for 30 days, button
  "Move to Trash" (FR-003)

## Phase 4: User Story 2 — Recover from the Trash within 30 days (P1)

- [ ] T012 [US2] `backend/internal/store/postgres/trash.go`: `ListTrash(collector, retention, cursor,
  limit)`, keyset on `(deleted_at, id)` descending, within the window, with `deleted_at` (FR-006)
- [ ] T013 [US2] Service `ListTrash` adds `purgeAt = deletedAt + retention`; `GET /api/trash` handler
  with the gallery's limit rules (FR-006)
- [ ] T014 [US2] `frontend/app/collection/trash/page.tsx` and `TrashList.tsx`: photograph, name, "Removed
  for good on <date>", Restore, empty state explaining 30 days, Load more; Trash link in
  `VaultHeader.tsx` (FR-006, US2/AC4)

## Phase 5: User Story 3 — Delete something for good, now (P2)

- [ ] T015 [US3] `Store.DeletePermanently(collector, id)` in `collectibles.go`: delete where
  `deleted_at IS NOT NULL`, `releaseImage` in the same transaction (FR-011)
- [ ] T016 [US3] Service `DeletePermanently` (log, drain); `DELETE /api/trash/{collectibleId}`, 204
  always (FR-011, FR-015)
- [ ] T017 [US3] Delete permanently in `TrashList.tsx` behind a native-dialog confirmation that names
  it, says it cannot be undone, focuses Cancel, and cancels on Enter (FR-011, 006 FR-023 pattern)

## Phase 6: Retention

- [ ] T018 `PurgeExpiredTrash(retention, limit)` in `trash.go`: lock up to 20 expired rows `FOR UPDATE
  SKIP LOCKED`, oldest deletion first; delete each and `releaseImage`; return the collector and
  collectible ids. Service wrapper logs each with identifiers only (FR-015); called after each deletion and at start-up in `main.go` before the drain (FR-010,
  FR-012, research Decision 4)

## Phase 7: Tests

- [ ] T019 `backend/tests/integration/trash_test.go`: a trashed collectible is absent from get, edit,
  list (every filter), the list count, statistics and both breakdowns (SC-003)
- [ ] T020 Same file: restore brings back attributes, photograph, version and gallery position; the
  rendition is fetchable by its owner while trashed and not by another collector (FR-007, FR-014)
- [ ] T021 Same file: deleting twice keeps the first `deleted_at`; restore of absent, other
  collector's, never-deleted and expired items is `ErrNotFound`; other collectors' Trash is empty
  (FR-004, FR-008, FR-013)
- [ ] T022 Same file: a retried add with the trashed collectible's submission key creates a new
  collectible and does not resurrect the old one (research Decision 5)
- [ ] T023 Same file: the listing order, window and pagination; the purge removes expired items and
  their photographs (files gone after the drain), is bounded, and leaves unexpired ones (FR-006,
  FR-009, FR-010, SC-004)
- [ ] T024 Same file: permanent deletion makes the rendition not-found at once and is idempotent
  (FR-011, SC-005)
- [ ] T025 Same file: restore racing the purge for items at expiry, repeatedly, staggered so both
  outcomes occur; each ends restored-and-present or destroyed-and-refused (FR-012)
- [ ] T026 `backend/tests/contract/trash_test.go`: the three endpoints' status codes and shapes against
  the OpenAPI document; 404 for another collector's item identical to absent (FR-013, SC-006)
- [ ] T027 [P] Frontend unit tests: the Undo notice restores and clears itself; the Trash list renders
  items, the empty state, restore, and the permanent-delete confirmation's default; the dialog's new
  copy

## Phase 8: Polish

- [ ] T028 Amend feature 006's spec (FR-020, FR-022, FR-023, SC-005) and research Decision 4 with a
  dated note pointing here (FR-017); update `delete_collectible.go`'s doc comment, which says there is
  no trash
- [ ] T029 Run the full gate: backend gofmt, vet (with and without `-tags integration`), build, unit,
  integration and contract suites; frontend lint, typecheck, unit tests, build; migration up, down and
  up again
- [ ] T030 Update the existing integration and contract tests that assert deletion destroys
  immediately (`delete_collectible_test.go`, `edit_delete_test.go`, `image_lifecycle_test.go`) to the
  new behaviour, keeping their intent where it still applies to permanent deletion

- [ ] T031 FR-018: the unique partial index in migration 000012; map its violation on add and edit to
  an `imageId` violation, "That photograph is already used by another collectible. Choose a different
  one."; integration test that adding with a trashed collectible's photograph is refused, and that
  destroying a trashed collectible never changes another collectible's photograph

## Dependencies

T001 → T004–T006, T012, T015, T018. T002 → T003 → T009. T004 → T019. Backend stories before their
frontend tasks. T030 after T005, since that change is what breaks them.
