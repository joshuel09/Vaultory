# Implementation Plan: Undo and a Recoverable Trash

**Branch**: `31-recoverable-trash` · **Spec**: [spec.md](spec.md) · **Issue**: #31
**Date**: 2026-10-25 · **Status**: planned

## Summary

A deleted collectible keeps its row and gains a `deleted_at`. Every vault read filters it out; three
new endpoints list, restore and permanently delete what is in the Trash. After 30 days a bounded
purge destroys trashed collectibles through feature 006's existing path — image row in the same
transaction, files queued and retried. The gallery offers Undo after a deletion; a Trash page offers
Restore and Delete permanently.

## Technical Context

| | |
|---|---|
| **Backend** | Go 1.26: store methods for trash, restore, list, permanent delete, purge; predicates on six existing queries |
| **Database** | PostgreSQL 17. Migration 000012 adds `collectibles.deleted_at` and two partial indexes |
| **API** | `GET /api/trash`, `POST /api/trash/{id}/restore`, `DELETE /api/trash/{id}`. `DELETE /api/collectibles/{id}` keeps its contract (204) and now trashes |
| **Frontend** | Next.js: dialog copy, Undo notice on the gallery, `/collection/trash` page, header link |
| **Testing** | Integration against PostgreSQL (every vault view hides the Trash, restore fidelity, window, purge, the restore/purge race, privacy); contract tests for the three endpoints; Vitest for the notice, Trash list and dialog |
| **Unknowns** | None. Six decisions in [research.md](research.md) |

## Constitution Check

| Principle | Assessment |
|---|---|
| **I. Collector-First** | Mistaken deletions become recoverable; the Trash shows photographs, not rows. Permanent deletion keeps 006's proportionate confirmation |
| **II. Separation** | Retention, restore eligibility and purge are Go and SQL rules. The frontend renders `purgeAt`, it never computes it |
| **III. Contract-First** | Three operations and one schema are added to `backend/api/openapi.yaml`, and frontend types are regenerated. The existing delete operation keeps its response shape, but its documented meaning changes; that is a deliberate, documented change |
| **IV. Data Integrity and Security** | Every Trash query carries `collector_id`. Another collector's Trash is indistinguishable from an empty one. 006 Decision 4 still governs destruction. The restore/purge race is settled by row locks. A reversible migration, whose down step destroys the Trash the way 006 destroyed deletions rather than resurrecting it |
| **V. Quality, Simplicity, Spec-Driven** | A column rather than a second table (Decision 1). Purge is opportunistic, as with 006 and 009. Edits are out of scope |

**Gate**: passes. **Re-check after design**: passes.

## Project Structure

```text
specs/010-recoverable-trash/
├── spec.md · plan.md · research.md · data-model.md · quickstart.md · tasks.md
├── contracts/trash.md
└── checklists/requirements.md

backend/migrations/000012_add_collectible_trash.{up,down}.sql
backend/internal/store/postgres/collectibles.go      deleted_at predicates; Trash, Restore, DeletePermanently
backend/internal/store/postgres/trash.go             ListTrash, PurgeExpiredTrash (new)
backend/internal/store/postgres/statistics.go        deleted_at predicates
backend/internal/collection/delete_collectible.go    Delete trashes; purge after
backend/internal/collection/trash.go                 Restore, ListTrash, DeletePermanently, PurgeExpiredTrash (new)
backend/internal/transport/httpapi/trash.go          three handlers (new); server.go routes; dto.go
backend/cmd/vaultory-api/main.go                     purge at start-up
backend/api/openapi.yaml                             three operations, TrashedCollectible, TrashPage
backend/tests/integration/trash_test.go              new
backend/tests/contract/trash_test.go                 new
frontend/lib/api/trash.ts                            new; lib/types/api.ts regenerated
frontend/components/collection/DeleteCollectibleDialog.tsx   copy (FR-003)
frontend/components/collection/EditCollectibleForm.tsx       land on the gallery with ?deleted=
frontend/components/collection/UndoDeleteNotice.tsx          new
frontend/components/collection/TrashList.tsx                 new (restore, delete permanently)
frontend/app/collection/trash/page.tsx                       new
frontend/app/collection/page.tsx, VaultHeader.tsx            notice, Trash link
specs/006-edit-delete-collectibles/spec.md                   amendments (FR-017)
```

## Merge order

Migration numbering assumes **#57 (feature 009, migration 000011) merges first**. golang-migrate
applies only versions above the current one, so if 000012 were applied before 000011 existed,
000011 would be skipped. The PR states this.

## Complexity Tracking

None.
