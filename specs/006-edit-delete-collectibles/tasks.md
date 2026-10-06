---

description: "Task list for feature 006, editing and deleting collectibles"
---

# Tasks: Edit & Delete Collectibles

**Input**: Design documents from `/specs/006-edit-delete-collectibles/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md),
[data-model.md](./data-model.md), [contracts/README.md](./contracts/README.md)

**Tests**: Included. The constitution requires automated tests for important business rules and
meaningful edge cases, and this feature's rules are mostly about what must *not* happen — a stale
save must not land, a photograph must not survive its collectible, another collector's entry must
not be reachable. None of those are visible without a test that tries them.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel — different files, no dependency on an incomplete task
- **[Story]**: US1, US2, US3 per spec.md
- Exact file paths are given for every task

## Path Conventions

Web application, per plan.md: `backend/` for Go, `frontend/` for Next.js. Backend tests live under
`backend/tests/{unit,integration,contract}/`; integration and contract suites are build-tagged and
need `-p 1` when run by hand.

---

## Phase 1: Setup — the contract and what is generated from it

**Purpose**: The OpenAPI document was written during planning and is already at 0.2.0. What remains
is propagating it, which must happen first because everything on the frontend is typed from it.

- [X] T001 Regenerate the frontend's API types with `cd frontend && npm run generate:api`, producing `frontend/lib/types/api.ts`
- [X] T002 Add the now-required `version` to the collectible factory in `frontend/tests/unit/fixtures.ts`
- [X] T003 [P] Re-export `Collectible` with its `version` and add `EditCollectibleRequest` to `frontend/lib/api/types.ts`
- [X] T004 Confirm `cd frontend && npm run typecheck && npm run lint` pass with the regenerated types

**Checkpoint**: the seam's types describe the three new operations; nothing implements them yet.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Everything more than one story needs. Two things in particular live here rather than in
a story, and the reason matters:

- **The image lifecycle** (migration 000010, the queue, the drain). The first operation that can
  detach a photograph is US1's edit, and the first that can orphan one wholesale is US2's delete.
  Putting it in either would make the other depend on it; putting it here keeps both independent.
  Leaving it out of the MVP is not an option — an edit that changes `image_id` without it leaves the
  old image fetchable, which is the defect FR-020 exists to prevent.
- **The edit screen shell**. US2's delete is reached from it (FR-022a), and US3 is an edit, so all
  three stories arrive through the same route.

**⚠️ CRITICAL**: no user story work begins until this phase is complete.

### Migrations

- [X] T005 Create `backend/migrations/000009_add_collectible_version.up.sql` adding `version integer NOT NULL DEFAULT 1` with a `>= 1` check, and `000009_add_collectible_version.down.sql` dropping both
- [X] T006 Create `backend/migrations/000010_create_pending_image_deletions.up.sql` per data-model.md — no foreign keys, with the comment explaining why — and `000010_create_pending_image_deletions.down.sql`
- [X] T007 Extend `backend/tests/integration/schema_test.go` to assert the `version` column, its check constraint, and the `pending_image_deletions` table and index exist

### Domain

- [X] T008 Add `Version int` to `Collectible` in `backend/internal/domain/collectible/collectible.go`
- [X] T009 Extract the per-field rules from `Draft.Validate` into a shared routine in `backend/internal/domain/collectible/collectible.go`, leaving `Draft.Validate` responsible only for the submission key plus that routine
- [X] T010 Create `backend/internal/domain/collectible/edit.go` with `EditDraft` (the same fields, `ExpectedVersion` in place of `SubmissionKey`) and a `Validate` that calls the shared routine
- [X] T011 [P] Extend `backend/tests/unit/collectible_validation_test.go` so every existing rule is asserted against both `Draft` and `EditDraft` — a table driven from one case list, so a rule cannot be added to one path only (SC-008) — and assert all sixteen status transitions are accepted, with none forbidden (FR-006)
- [X] T012 Add a unit test in `backend/tests/unit/collectible_validation_test.go` that `EditDraft` rejects a missing or zero `ExpectedVersion` and that `Draft` still requires a submission key

### Store

- [X] T013 Add `c.version` to `collectibleColumns` and to `scanRow` in `backend/internal/store/postgres/collectibles.go`
- [X] T014 Add `Store.Get(ctx, collectorID, collectibleID) (Row, error)` to `backend/internal/store/postgres/collectibles.go`, carrying `collector_id` as a predicate and returning `ErrNotFound` for absent or another collector's
- [X] T015 Create `backend/internal/store/postgres/image_deletions.go` with `QueueImageDeletion` (inside a caller's transaction), `PendingImageDeletions(limit)` and `ForgetImageDeletion`
- [X] T016 Add an unexported `releaseImage(ctx, tx, collectorID, imageID)` helper to `backend/internal/store/postgres/collectibles.go` that deletes the `collectible_images` row and queues its storage keys in one transaction

### Service

- [X] T017 Create `backend/internal/collection/get_collectible.go` with `Service.Get`
- [X] T018 Create `backend/internal/collection/image_cleanup.go` with `Service.DrainImageDeletions(ctx, limit)` — delete each file, treat an already-absent file as deleted, remove the queue row, and leave a row behind on failure
- [X] T019 Call `DrainImageDeletions` once at start-up in `backend/cmd/vaultory-api/main.go`, logging the outcome and never failing start-up on it

### Transport

- [X] T020 Add `Version int` to `collectibleResponse` and populate it in `toCollectibleResponse` in `backend/internal/transport/httpapi/dto.go`
- [X] T021 Add `handleGetCollectible` to `backend/internal/transport/httpapi/collectibles.go` — parse the id, 404 on a malformed one, 404 on `ErrNotFound`
- [X] T022 Register `GET /api/collectibles/{collectibleId}` in `backend/internal/transport/httpapi/server.go` and update the comment that currently says there is no collectible-detail endpoint

### Frontend shell

- [X] T023 Add `getCollectible(id)` to `frontend/lib/api/collectibles.ts`
- [X] T024 Extract the form body of `frontend/components/collection/AddCollectibleForm.tsx` into a new `frontend/components/collection/CollectibleForm.tsx` taking initial values, an existing image, a submit label and an `onSubmit`
- [X] T025 Rewrite `frontend/components/collection/AddCollectibleForm.tsx` as a thin wrapper over `CollectibleForm` that owns the submission key and calls `addCollectible`, preserving the no-`router.refresh()` comment and its reasoning
- [X] T026 Confirm `frontend/tests/unit/add-collectible-form.test.tsx` still passes unchanged against the wrapper; adjust only selectors if the DOM moved, not assertions
- [X] T027 Create `frontend/app/collection/[id]/edit/page.tsx` as a Server Component that forwards the request's cookies to `VAULTORY_BACKEND_ORIGIN`, redirects to `/sign-in?next=…` on 401, and calls `notFound()` on 404 (research.md Decision 13)
- [X] T028 [P] Create `frontend/app/collection/[id]/edit/loading.tsx` and `frontend/app/collection/[id]/edit/not-found.tsx` as designed states matching `frontend/app/collection/loading.tsx`
- [X] T029 Add an edit affordance to `frontend/components/collection/CollectibleCard.tsx` linking to `/collection/{id}/edit`, carrying the active status filter as a `next` parameter
- [X] T030 [P] Add a contract test for `getCollectible` in `backend/tests/contract/edit_delete_test.go` — 200 for the owner with every field and a `version`, 404 for another collector, 404 for a random UUID, byte-identical bodies for the last two, and 401 with no collection content when no session is presented (FR-033)
- [X] T031 [P] Add an integration test in `backend/tests/integration/image_lifecycle_test.go` that `DrainImageDeletions` removes queued files and their rows, and that a failing store leaves the row for a later drain (FR-020a)

**Checkpoint**: a collector can open any of their collectibles on an edit screen and see its values. Nothing can be changed yet.

---

## Phase 3: User Story 1 — Correct a collectible already in my vault (Priority: P1) 🎯 MVP

**Goal**: A collector changes any attribute of a collectible they own and the change is saved,
validated exactly as adding is, refused if someone changed it first, and invisible to the gallery's
ordering.

**Independent Test**: Add a collectible, add a second one after it, edit the first one's name and
status, reload. The new values show, there is still one entry, and it has not moved to the front.

### Tests for User Story 1

> Write these first and watch them fail.

- [X] T032 [P] [US1] Contract test for `editCollectible` in `backend/tests/contract/edit_delete_test.go` — 200 with the new version, 400 carrying every violation together, 404 for another collector's, 409 for a stale version, 401 with no session, and a collector id supplied in a header, body field or query parameter ignored entirely (FR-032, FR-033)
- [X] T033 [P] [US1] Integration test in `backend/tests/integration/edit_collectible_test.go` that an edit changes exactly one row, leaves identical siblings untouched, creates nothing, and raises `version` by one (FR-007, FR-008), and that an edit failing partway leaves the collectible exactly as it was — no field changed, no version bump (FR-009)
- [X] T034 [US1] Integration test in `backend/tests/integration/edit_collectible_test.go` that every optional attribute can be cleared back to NULL, and that a cleared purchase price is NULL rather than `0.00` (FR-005)
- [X] T035 [P] [US1] Integration test in `backend/tests/integration/version_conflict_test.go` that a second edit carrying the first's version is refused with the current collectible and writes nothing, and that two concurrent edits produce one winner and one refusal (FR-027)
- [X] T036 [P] [US1] Integration test in `backend/tests/integration/ownership_test.go` that reading and editing another collector's collectible both answer not-found, and that the response is identical to one for a non-existent id (FR-030, FR-031, SC-002)
- [X] T037 [P] [US1] Integration test in `backend/tests/integration/money_roundtrip_test.go` that reading a purchase price and saving it back unchanged leaves it byte-identical, across the amounts already covered for adding (FR-013)
- [X] T038 [P] [US1] Integration test in `backend/tests/integration/ordering_test.go` that a collectible edited after a later one was added keeps its gallery position (FR-028)
- [X] T039 [US1] Integration test in `backend/tests/integration/edit_collectible_test.go` that an edit changing only the name, sending the current `imageId` back, keeps the photograph and does not queue its files (FR-018) — the sharpest edge in the contract

### Implementation for User Story 1

- [X] T040 [US1] Add `Store.Edit` to `backend/internal/store/postgres/collectibles.go` — `SELECT … WHERE id AND collector_id FOR UPDATE`, then `ErrNotFound` / `ErrVersionConflict` / `UPDATE … SET version = version + 1`, releasing a replaced or removed image through `releaseImage` in the same transaction (research.md Decision 6)
- [X] T041 [US1] Add `ErrVersionConflict` to `backend/internal/store/postgres/collectibles.go`, carrying the current row so the caller need not re-read it
- [X] T042 [US1] Create `backend/internal/collection/edit_collectible.go` with `Service.Edit` — validate, check image ownership for a field-level message, apply, then drain a bounded batch of pending image deletions
- [X] T043 [US1] Add `editCollectibleRequest` and its `toEditDraft` to `backend/internal/transport/httpapi/dto.go`, mirroring `EditCollectibleRequest` in the contract exactly
- [X] T044 [US1] Add `CodeVersionConflict` and `WriteVersionConflict` to `backend/internal/transport/httpapi/errors.go`, rendering the 409 body defined by `VersionConflictResponse`
- [X] T045 [US1] Add `handleEditCollectible` to `backend/internal/transport/httpapi/collectibles.go` and register `PUT /api/collectibles/{collectibleId}` in `backend/internal/transport/httpapi/server.go`
- [X] T046 [US1] Add `editCollectible(id, body)` to `frontend/lib/api/collectibles.ts`
- [X] T047 [US1] Teach `frontend/lib/api/errors.ts` to recognise `version_conflict` and expose the `current` collectible it carries
- [X] T048 [US1] Create `frontend/components/collection/EditCollectibleForm.tsx` wrapping `CollectibleForm`, holding the version, echoing the current `imageId` when the photograph is untouched, and returning through `safeRedirect` with a single `router.push` (research.md Decision 14). It MUST refuse a second submission while one is in flight (FR-036): two `PUT`s carrying the same `expectedVersion` means the second comes back 409 and tells the collector the collectible changed since they opened it — about their own save
- [X] T049 [US1] Create `frontend/components/collection/VersionConflictNotice.tsx` — says the collectible changed, shows how it now reads, and offers to load the current values rather than silently replacing what the collector typed
- [X] T050 [P] [US1] Unit test `frontend/tests/unit/collectible-form.test.tsx` — initial values populate and attributes never supplied render empty rather than defaulted (FR-003), clearing an optional field submits it as absent, field errors render against the right inputs, entered values survive a failed save (FR-035), and a double-clicked save issues exactly one request (FR-036)
- [ ] T051 [P] [US1] End-to-end test `frontend/tests/e2e/edit-collectible.spec.ts` covering the independent test above plus a validation failure and recovery

**Checkpoint**: a collection is maintainable. This is a shippable increment on its own.

---

## Phase 4: User Story 2 — Remove a collectible from my vault (Priority: P2)

**Goal**: A collector deletes a collectible permanently, behind a confirmation proportionate to the
fact that it cannot be undone, with the photograph going too.

**Independent Test**: Add two identical collectibles, delete one through the confirmation, reload.
Exactly one remains. Cancelling the confirmation deletes nothing.

### Tests for User Story 2

- [ ] T052 [P] [US2] Contract test for `deleteCollectible` in `backend/tests/contract/edit_delete_test.go` — 204 when it existed, 204 when repeated, 204 for a random UUID, 204 for another collector's with nothing deleted, and identical responses throughout (FR-025, FR-031); plus 401 with no session and an asserted collector id ignored — the destructive route is the worst place for an unproven auth path, and its correct answer is a silent 204 (FR-032, FR-033)
- [ ] T053 [P] [US2] Integration test in `backend/tests/integration/delete_collectible_test.go` that deleting removes exactly one row, leaves identical siblings untouched, and that the entry is absent from the gallery and from every status filter afterwards (FR-024, FR-026)
- [ ] T054 [US2] Integration test in `backend/tests/integration/delete_collectible_test.go` that deleting a collectible added within the idempotency window succeeds and cascades its submission row, and that replaying that add afterwards creates a new collectible rather than failing
- [ ] T055 [P] [US2] Integration test in `backend/tests/integration/image_lifecycle_test.go` that deleting a collectible with a photograph removes the `collectible_images` row in the same transaction, queues both storage keys, and makes the rendition answer not-found immediately (FR-020)
- [ ] T056 [US2] Integration test in `backend/tests/integration/delete_collectible_test.go` that a deletion emits its record exactly once and only when a row was actually deleted, carrying identifiers and no collection content (FR-040, FR-041)

### Implementation for User Story 2

- [ ] T057 [US2] Add `Store.Delete` to `backend/internal/store/postgres/collectibles.go` — `DELETE … WHERE id AND collector_id RETURNING image_id`, releasing the image through `releaseImage` in the same transaction, reporting whether a row was removed
- [ ] T058 [US2] Create `backend/internal/collection/delete_collectible.go` with `Service.Delete` — delete, emit the structured deletion event only when a row was removed, then drain a bounded batch (research.md Decision 8)
- [ ] T059 [US2] Add `handleDeleteCollectible` to `backend/internal/transport/httpapi/collectibles.go` returning 204 in every case including a malformed id, and register `DELETE /api/collectibles/{collectibleId}` in `backend/internal/transport/httpapi/server.go`
- [ ] T060 [US2] Add `deleteCollectible(id)` to `frontend/lib/api/collectibles.ts`
- [ ] T061 [US2] Create `frontend/components/collection/DeleteCollectibleDialog.tsx` using a native `<dialog>` opened with `showModal()` — names the collectible, states that deletion cannot be undone, focuses Cancel on open, and makes the destructive button neither the autofocused control nor the dialog's default submit, and refuses a second confirmation while one is in flight (FR-023, FR-036, FR-037)
- [ ] T062 [US2] Add the delete affordance to `frontend/components/collection/EditCollectibleForm.tsx` only, never to `CollectibleCard.tsx` (FR-022a)
- [ ] T063 [P] [US2] Unit test `frontend/tests/unit/delete-dialog.test.tsx` — Escape closes without deleting, Enter on open closes without deleting, Cancel holds initial focus, the collectible's name appears in the dialog, focus returns to the opener on close, and a double-clicked confirm issues exactly one request (FR-036)
- [ ] T064 [P] [US2] End-to-end test `frontend/tests/e2e/delete-collectible.spec.ts` covering the independent test, cancellation, deleting the last collectible into the empty state, deleting while a status filter is active, and that no gallery entry offers a delete control (FR-022a)

**Checkpoint**: duplicates and unwanted entries can be removed. Feature 001's unrecoverable-duplicate defect class is closed.

---

## Phase 5: User Story 3 — Change or remove a collectible's photograph (Priority: P3)

**Goal**: A collector replaces a photograph, or removes it so the designed placeholder returns, with
the old photograph no longer retrievable by anyone.

**Independent Test**: Add a collectible with a photograph, replace it, confirm the gallery shows the
new one and the old rendition URL answers not-found. Remove it and confirm the placeholder.

### Tests for User Story 3

- [ ] T065 [P] [US3] Integration test in `backend/tests/integration/image_lifecycle_test.go` that replacing a photograph deletes the old image row, queues its files, leaves the new one intact, and makes only the old rendition answer not-found (FR-020)
- [ ] T066 [US3] Integration test in `backend/tests/integration/image_lifecycle_test.go` that omitting `imageId` sets `image_id` to NULL and releases the image, and that the collectible then renders without one (FR-017)
- [ ] T067 [P] [US3] Integration test in `backend/tests/integration/image_ownership_constraint_test.go` that an edit referencing another collector's image is refused as an unknown image, in the same words as one that never existed, and that the composite foreign key refuses it even if the check is bypassed (FR-021, FR-031)
- [ ] T068 [P] [US3] Integration test in `backend/tests/integration/edit_collectible_test.go` that a refused edit — a validation failure or a version conflict — leaves the existing photograph in place and queues nothing (FR-019)

### Implementation for User Story 3

- [ ] T069 [US3] Wire `frontend/components/collection/ImagePicker.tsx` into `CollectibleForm.tsx` so it opens with the collectible's existing photograph and its Remove control clears the reference rather than only the preview
- [ ] T070 [US3] Ensure `frontend/components/collection/EditCollectibleForm.tsx` sends `imageId: null` when the collector removed the photograph and the current id when they did not touch it, with a comment naming the consequence of getting it wrong
- [ ] T071 [P] [US3] End-to-end test `frontend/tests/e2e/edit-photograph.spec.ts` — replace, remove, a refused oversized replacement leaving the original in place, and the placeholder after removal

**Checkpoint**: all three stories work independently.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [ ] T072 [P] Extend `frontend/tests/e2e/accessibility.spec.ts` with the edit screen and the delete dialog — keyboard reachable from the gallery, focus trapped, focus restored (FR-037)
- [ ] T073 [P] Extend `frontend/tests/e2e/appearance.spec.ts` so the edit screen and the dialog are asserted in both dark and light (FR-038)
- [ ] T074 [P] Extend `frontend/tests/e2e/responsive.spec.ts` with the edit screen and the dialog at tablet and mobile widths (FR-039)
- [ ] T075 [P] Add an integration test in `backend/tests/integration/query_count_test.go` that an edit and a delete each issue a bounded number of statements and introduce no per-entry query
- [ ] T076 Update the Current features list in `CLAUDE.md` with feature 006 and its outcome
- [ ] T077 Update `README.md` where it describes what a collector can do, so editing and deleting are not missing from the product description
- [ ] T078 Run every suite in full — `make test` (backend), `cd frontend && npm run test` (frontend units, which `make test` does not cover), and `make test-e2e` — and record genuine results, including any flakiness, rather than rounding to green
- [ ] T079 Walk `specs/006-edit-delete-collectibles/quickstart.md` end to end against a running stack, including the curl scenarios for another collector's collectible
- [ ] T080 Confirm the production build gates pass: `cd frontend && npm run build` and `make prod-build`. The constitution lists both as MUST before a feature is complete, and neither is reached by `make test`

---

## Dependencies & Execution Order

### Phase dependencies

- **Phase 1 (Setup)**: no dependencies. Everything typed on the frontend waits on it.
- **Phase 2 (Foundational)**: depends on Phase 1 for the generated types only; the backend half can
  begin immediately. **Blocks all three stories.**
- **Phase 3–5 (Stories)**: all depend on Phase 2. US1 is the MVP; US2 and US3 are independent of each
  other and of US1 once Phase 2 is done, because the image lifecycle and the edit screen they share
  are both foundational.
- **Phase 6 (Polish)**: depends on whichever stories are being shipped. T078, T079 and T080 are the quality gates and come last.

### Within Phase 2

Migrations (T005–T007) → store (T013–T016) → service (T017–T019) → transport (T020–T022). The domain
tasks (T008–T012) and the frontend shell (T023–T029) run alongside that chain, not after it.

T024 (extracting `CollectibleForm`) blocks T025, T027 and every form task in Phases 3–5. It is the
one frontend task worth doing early and carefully.

### Within each story

Tests before implementation. Store before service before transport. Frontend after the API client.

### Parallel opportunities

- T002, T003 together; T011, T012 together.
- All of Phase 2's test tasks (T030, T031) alongside its implementation.
- Every test task in a story phase is marked [P] — they touch different files and are written before
  the implementation they describe.
- US2 and US3 can be built by different people at the same time once Phase 2 is done.
- All of Phase 6 except T078, T079 and T080, which need everything else finished.

---

## Parallel Example: User Story 1

```bash
# The tests, together, before any of the implementation:
T032  Contract test for editCollectible         backend/tests/contract/edit_delete_test.go
T033  Edit touches exactly one row              backend/tests/integration/edit_collectible_test.go
T035  Stale version refused, nothing written    backend/tests/integration/version_conflict_test.go
T036  Another collector's answers not-found     backend/tests/integration/ownership_test.go
T037  Price survives a read-modify-write        backend/tests/integration/money_roundtrip_test.go
T038  An edit does not move the collectible     backend/tests/integration/ordering_test.go

# Then the implementation chain, which is sequential:
T040 → T041 → T042 → T043 → T044 → T045
```

---

## Implementation Strategy

### MVP — User Story 1 only

1. Phase 1 (T001–T004)
2. Phase 2 (T005–T031) — the whole of it, including the image lifecycle
3. Phase 3 (T032–T051)
4. **Stop and validate**: Scenarios 1, 2, 3, 6 and 8 of `quickstart.md`
5. Shippable: a collection that can be corrected

### Incremental delivery

1. Setup + Foundational → a collectible can be opened for editing
2. US1 → corrections work → demo
3. US2 → deletion works → demo
4. US3 → photographs can be changed → demo
5. Polish → ship

### What to be careful about

Three tasks are where this feature will go wrong if it goes wrong, and each has a test that exists
specifically to catch it:

- **T039** — an edit that omits `imageId` destroys the photograph. The contract is a full
  replacement; the form must echo the current id back.
- **T040** — using `UPDATE … WHERE version = $n` instead of `SELECT … FOR UPDATE` collapses 404 and
  409 into one indistinguishable answer, and SC-002 and SC-006 both stop being provable.
- **T055 / T065** — deleting or replacing while leaving the `collectible_images` row behind looks
  correct in the product and leaves the photograph fetchable by its owner indefinitely.

---

## Notes

- [P] means a different file and no dependency on an unfinished task.
- Run the backend suites with `-p 1` by hand; the integration and contract packages share one database.
- Commit per task or per logical group, and push at every checkpoint — a local commit is invisible.
- 80 tasks: 4 setup, 27 foundational, 20 for US1, 13 for US2, 7 for US3, 9 polish.
