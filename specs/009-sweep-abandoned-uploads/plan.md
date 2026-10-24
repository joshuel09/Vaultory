# Implementation Plan: Sweep Abandoned Uploads

**Branch**: `30-sweep-abandoned-uploads` · **Spec**: [spec.md](spec.md) · **Issue**: #30
**Date**: 2026-10-24 · **Status**: planned

## Summary

One store method finds unreferenced uploads older than 30 days, and in a single transaction deletes
their rows and queues their files in feature 006's `pending_image_deletions`. The existing drain then
deletes the files. The service runs the sweep after each successful upload and once at start-up.
**Row locks, not timing, decide every race with a save** (research Decision 2). The one visible
change is the photograph field finally showing its own error, with a message asking for the
photograph to be uploaded again.

## Technical Context

| | |
|---|---|
| **Backend** | Go 1.26: one store method, one service method, two call sites, one shared message |
| **Database** | PostgreSQL 17. Migration 000011 adds two indexes; no table changes |
| **Frontend** | `ImagePicker` gains an `error` prop; `CollectibleForm` passes `fieldErrors.imageId` |
| **Contract** | Unchanged. `imageId` violations already exist; only their message text changes |
| **Testing** | Integration tests against real PostgreSQL (eligibility, the race, retries, the refused save); Vitest for the field error |
| **Unknowns** | None. Four decisions in [research.md](research.md) |

## Constitution Check

| Principle | Assessment |
|---|---|
| **I. Collector-First** | The rare refused save says what happened and what to do, on the field it concerns, without clearing anything the collector typed |
| **II. Separation** | Eligibility, locking and queueing live in Go and SQL. The frontend only renders a message it is given |
| **III. Contract-First** | The OpenAPI contract is unchanged: same field, same code; only the message text differs |
| **IV. Data Integrity and Security** | The whole point. Deleting the row is the privacy guarantee and happens transactionally with queueing the files, so they cannot disagree. A saved collectible cannot lose its photograph (FR-006), and that is enforced by row locks the database holds. A reversible migration adds the indexes |
| **V. Quality, Simplicity, Spec-Driven** | Reuses 006's queue and drain rather than a second path. No ticker: the same reasoning as 006 Decision 5. The race is tested, not argued |

**Gate**: passes. **Re-check after design**: passes.

## Project Structure

```text
specs/009-sweep-abandoned-uploads/
├── spec.md · plan.md · research.md · quickstart.md · tasks.md
└── checklists/requirements.md
```

No `data-model.md`: no entity changes. No `contracts/`: the API is unchanged.

```text
backend/migrations/000011_index_image_sweep.{up,down}.sql      new
backend/internal/store/postgres/images.go                      + SweepAbandonedImages
backend/internal/collection/image_cleanup.go                   + SweepAbandonedUploads, grace constant
backend/internal/collection/upload_image.go                    sweep after a successful upload
backend/internal/collection/{add,edit}_collectible.go          shared "upload it again" message
backend/cmd/vaultory-api/main.go                               sweep at start-up, before the drain
backend/tests/integration/abandoned_upload_sweep_test.go       new
backend/tests/integration/{ownership,image_ownership_constraint}_test.go   message assertions
frontend/components/collection/ImagePicker.tsx                 + error prop
frontend/components/collection/CollectibleForm.tsx             pass fieldErrors.imageId
frontend/tests/unit/collectible-form.test.tsx                  the field error renders
```

## Complexity Tracking

None.
