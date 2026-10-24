# Tasks: Sweep Abandoned Uploads

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [quickstart.md](quickstart.md)

**Tests**: required by the constitution for important business rules. The race (FR-006) is the one
that matters most, and it can only be shown against real PostgreSQL.

**Format**: `[ID] [P?] [Story] Description`.

## Phase 1: Setup

- [ ] T001 Add `backend/migrations/000011_index_image_sweep.up.sql` creating
  `collectible_images_created_idx ON collectible_images (created_at)` and
  `collectibles_image_idx ON collectibles (image_id)`, and the matching `.down.sql` dropping both
  (research Decision 4)

## Phase 2: Foundational

- [ ] T002 Add `SweepAbandonedImages(ctx, olderThan time.Duration, limit int) (int, error)` to
  `backend/internal/store/postgres/images.go`: one transaction; lock eligible rows
  `FOR UPDATE SKIP LOCKED`, oldest first, bounded; then delete those ids again conditioned on no
  collectible referencing them, inserting their keys into `pending_image_deletions` in the same
  statement; return the number removed (FR-001, FR-003–FR-006, FR-008, research Decision 2)
- [ ] T003 Add `abandonedUploadGrace = 30 * 24 * time.Hour` and
  `SweepAbandonedUploads(ctx) int` to `backend/internal/collection/image_cleanup.go`: one bounded
  batch through T002, log the count when non-zero (FR-011), log and swallow errors (FR-010), then
  `DrainImageDeletions` (FR-002, FR-003)

## Phase 3: User Story 1 — Abandoned uploads stop consuming storage (P1) 🎯 MVP

**Independent test**: quickstart A and D.

- [ ] T004 [US1] Call `SweepAbandonedUploads` at the end of a successful `UploadImage` in
  `backend/internal/collection/upload_image.go`, after the image row is inserted, never affecting
  the returned image or error (FR-009, FR-010)
- [ ] T005 [US1] Call `SweepAbandonedUploads` at start-up in `backend/cmd/vaultory-api/main.go`
  before the existing drain, with the same not-fatal treatment (FR-009)
- [ ] T006 [US1] In `backend/tests/integration/abandoned_upload_sweep_test.go`: an upload backdated
  past the grace period is swept, has a queue entry, and both files are gone after the drain; a
  second, older upload replaced before saving is swept while the saved one survives (US1/AC1, AC2)
- [ ] T007 [US1] In the same file: with an image store whose deletes fail, the queue entry survives
  the drain, and a later drain with a working store removes the files (US1/AC3, SC-004)
- [ ] T008 [US1] In the same file: a sweep with more eligible uploads than its bound removes exactly
  the bound, oldest first (FR-008, SC-001)

## Phase 4: User Story 2 — An in-progress form never loses its photograph (P1)

**Independent test**: quickstart B and C.

- [ ] T009 [US2] In `backend/tests/integration/abandoned_upload_sweep_test.go`: an upload one hour
  younger than the grace period survives; an upload older than it that a collectible references
  survives (US2/AC1, AC2, FR-004, FR-005)
- [ ] T010 [US2] In the same file: the race. Repeatedly start an add referencing an eligible upload
  and a sweep together; every iteration ends with the collectible saved with its photograph and the
  upload present, or the add refused with the image violation and the upload gone. Never a saved
  collectible with a null image (US2/AC3, FR-006, SC-002)
- [ ] T011 [US2] In the same file: the same race with an edit that attaches the eligible upload to an
  existing collectible (FR-006, edge case "an edit replaces a photograph")

## Phase 5: User Story 3 — A form left open past the grace period fails clearly (P2)

**Independent test**: quickstart E, and T012.

- [ ] T012 [US3] Replace the four "That image could not be found." messages in
  `backend/internal/collection/add_collectible.go` and `edit_collectible.go` with one shared
  constant: "That photograph is no longer available. Upload it again, or save without one." Same for
  an absent image and someone else's, as FR-027 of feature 001 requires (FR-007)
- [ ] T013 [US3] Update the message assertions in `backend/tests/integration/ownership_test.go` and
  `image_ownership_constraint_test.go`, and add to the sweep test file: adding with a swept upload
  is refused with that message on `imageId` and creates nothing (US3/AC1)
- [ ] T014 [P] [US3] Give `ImagePicker` in `frontend/components/collection/ImagePicker.tsx` an
  optional `error` prop rendered as the field's alert, and pass `fieldErrors.imageId` from
  `frontend/components/collection/CollectibleForm.tsx` (FR-007)
- [ ] T015 [US3] In `frontend/tests/unit/collectible-form.test.tsx`: an `imageId` violation renders on
  the photograph field and the typed values remain (FR-007)

## Phase 6: Polish

- [ ] T016 Run the full gate: backend `go vet` (with and without `-tags integration`), build, unit,
  integration and contract suites against PostgreSQL; frontend lint, typecheck, unit tests, build
- [ ] T017 Confirm migration 000011 applies, rolls back, and reapplies cleanly
- [ ] T018 Record in `docs/deferred.md` that "Reclaiming unreferenced uploads" is now done by
  feature 009, if that file lists it

## Dependencies

T001 → T002 → T003 → T004, T005. Tests T006–T011 need T002–T003. T012 → T013. T014 → T015.
T014 touches only frontend files and can run alongside any backend task. T016–T018 last.

## Implementation strategy

MVP is US1 plus US2 together: a sweeper is not shippable without the race guarantee. US3 is the
clear failure for the rare form left open past the grace period.
