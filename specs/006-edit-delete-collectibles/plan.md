# Implementation Plan: Edit & Delete Collectibles

**Branch**: `26-edit-delete-collectibles` | **Date**: 2026-10-01 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/006-edit-delete-collectibles/spec.md`

## Summary

A collector can change any attribute of a collectible already in their vault, replace or remove its
photograph, and delete it permanently behind a confirmation that names it. Feature 001 shipped
adding and browsing and excluded both; its own spec recorded the cost, since the submission-key
defence exists to close a defect that would otherwise be unrecoverable.

Technically this is the first time Vaultory writes over a row rather than only inserting one, and
almost every risk in the feature follows from that:

- **A version column** (`collectibles.version`) makes a stale save refusable instead of silently
  destructive. It is the one schema change the feature genuinely needs.
- **`SELECT … FOR UPDATE` inside the edit transaction** keeps "not yours" (404) distinguishable from
  "out of date" (409). A conditional `UPDATE` would collapse them.
- **Deleting the `collectible_images` row in-transaction** is what makes a photograph unfetchable.
  Unlinking it is not enough — renditions are authorised on the image's own owner, because the
  upload preview has to work before any collectible references it.
- **A small queue table** carries the storage keys of files whose rows are gone, so their removal can
  be retried without ever being able to block a deletion.
- **`DELETE` answers 204 unconditionally**, which makes retries safe and makes another collector's
  identifier indistinguishable from a fictional one.

On the frontend, `AddCollectibleForm` becomes a shared `CollectibleForm` driven by initial values and
a submit action, so the add and edit screens cannot drift in what they validate or how they recover
from a failed save. The confirmation is a native `<dialog>`; no dialog dependency is added.

Full reasoning for each decision, including what was rejected, is in [research.md](./research.md).

## Technical Context

**Language/Version**: Go 1.26 (backend), TypeScript 5.x on Next.js 15.5 App Router (frontend)

**Primary Dependencies**: Go standard library `net/http`, `jackc/pgx/v5`, `google/uuid`,
`golang-migrate`; React 19, Tailwind CSS, shadcn/ui foundations. **No new dependency in either
application** — the confirmation dialog uses the platform `<dialog>` element (research.md Decision 9).

**Storage**: PostgreSQL 17. Two migrations: `000009` adds `collectibles.version`, `000010` creates
`pending_image_deletions`. Both reversible. Image bytes stay in the filesystem image store behind
`imagestore.Store`.

**Testing**: `go test` — unit (no database), integration and contract (tagged, real database, `-p 1`);
Vitest for frontend units; Playwright across desktop, tablet and mobile projects.

**Target Platform**: Linux containers under Docker Compose; browsers from mobile to desktop.

**Project Type**: Web application — separate Go service and Next.js application, REST between them.

**Performance Goals**: An edit and a delete are each one transaction and a bounded number of
statements; neither introduces a per-entry query. The queue drain is bounded per request (a small
batch) so it cannot turn a delete into an unbounded operation.

**Constraints**: Monetary amounts stay exact decimals end to end — bound as strings, cast in SQL,
never a JSON number (Principle IV). Every collection statement carries `collector_id` as a
predicate. No 403 anywhere in the transport.

**Scale/Scope**: Three new API operations, two migrations, one new frontend route, one form
generalised, one dialog. Roughly 30 new or changed source files plus tests.

## Constitution Check

*GATE: passed before Phase 0, re-checked after Phase 1 design. No violations; the Complexity Tracking
table is therefore absent rather than empty.*

### I. Collector-First Experience

- Editing reuses the gallery's visual language and the existing form components; the edit screen
  mirrors the add screen rather than becoming a settings page.
- Loading, empty, validation, success and error states are required for both new screens (FR-034 to
  FR-036) and the version-conflict state is designed, not a raw error.
- Dark and light both covered, following system preference (FR-038), including the dialog.
- The confirmation is deliberate but not punitive: it names the collectible, it is not dismissible
  into a deletion, and it does not demand a typed phrase (research.md Decision 9).
- Accessibility is in the requirements, not after them: focus into the dialog, Escape to abort, focus
  restored on close, full keyboard operation (FR-037).

### II. Clear Architecture and Separation of Responsibilities

- The domain decides validity (`EditDraft.Validate`), the store decides how rows change, the
  `collection` service decides the order of operations, and `httpapi` renders. Unchanged shape.
- Nothing new lives only in the frontend. Notably, "deleting twice is not an error" is enforced in Go
  as a 204 rather than as a 404 the browser reinterprets (research.md Decision 7).
- The queue drain is a service concern, not an HTTP one; the transport does not know it exists.

### III. Contract-First and Type-Safe Development

- The three operations are added to the OpenAPI document before any handler is written, and the
  frontend's types are regenerated from it. No response shape is hand-written.
- The version crosses as a typed schema field rather than an `ETag`, specifically so the generator
  covers it (research.md Decision 2).
- A breaking change is documented: `Collectible` gains a required `version`, and the document goes to
  0.2.0.

### IV. Data Integrity and Security

- Both schema changes ship as reversible migrations with working `down` files.
- **"MUST NOT be silently lost, overwritten, or corrupted"** is the principle this whole feature turns
  on, and it is why a stale save is refused rather than applied (FR-027).
- Every statement carries `collector_id`; another collector's collectible answers 404 on read and
  edit, and 204 on delete, with nothing deleted — both indistinguishable from a non-existent id.
- The composite foreign key continues to make a cross-collector image reference impossible in the
  database, not merely in application code.
- Amounts remain exact decimals across the read-modify-write cycle, which is a new exposure: feature
  001 only ever wrote an amount it had just parsed. An integration test reads a price, saves it back
  untouched, and asserts it is byte-identical (FR-013).
- The deletion record carries identifiers only and never collection content (FR-040).

### V. Quality, Simplicity, and Spec-Driven Development

- Written spec first, clarified, every task traceable to a requirement.
- One refactor is included — `AddCollectibleForm` → shared `CollectibleForm` — and it is justified:
  SC-008 requires the two paths to validate identically, and two 290-line copies would make that a
  promise rather than a property. No other refactoring rides along.
- No new dependency in either application.
- No `updated_at` column, no background ticker, no audit table: each was considered and rejected as
  machinery the requirements do not ask for (research.md Decisions 1, 5, 8).

## Project Structure

### Documentation (this feature)

```text
specs/006-edit-delete-collectibles/
├── plan.md              # This file
├── research.md          # Phase 0 — 14 decisions with rejected alternatives
├── data-model.md        # Phase 1 — migrations, transactions, entities
├── quickstart.md        # Phase 1 — how to prove it works
├── contracts/
│   └── README.md        # Points at the real contract and records what 0.2.0 added
├── checklists/
│   └── requirements.md  # Spec quality, 16/16
├── spec.md
└── tasks.md             # /speckit-tasks output — not created by /speckit-plan
```

### Source Code (repository root)

```text
backend/
├── migrations/
│   ├── 000009_add_collectible_version.{up,down}.sql      # NEW
│   └── 000010_create_pending_image_deletions.{up,down}.sql  # NEW
├── internal/
│   ├── domain/collectible/
│   │   ├── collectible.go        # CHANGED: Version field, EditDraft, shared field validation
│   │   └── edit.go               # NEW: EditDraft and its Validate
│   ├── collection/
│   │   ├── get_collectible.go    # NEW: read one for editing
│   │   ├── edit_collectible.go   # NEW: validate, check image ownership, apply
│   │   ├── delete_collectible.go # NEW: delete, queue image files, log the event
│   │   └── image_cleanup.go      # NEW: drain the pending-deletion queue
│   ├── store/postgres/
│   │   ├── collectibles.go       # CHANGED: Get, Edit (FOR UPDATE), Delete, version in projection
│   │   └── image_deletions.go    # NEW: queue reads and writes
│   └── transport/httpapi/
│       ├── server.go             # CHANGED: three routes
│       ├── collectibles.go       # CHANGED: three handlers
│       ├── dto.go                # CHANGED: edit request, version, conflict body
│       └── errors.go             # CHANGED: version_conflict code and writer
└── tests/
    ├── unit/collectible_validation_test.go   # CHANGED: parity between add and edit
    ├── integration/edit_collectible_test.go  # NEW
    ├── integration/delete_collectible_test.go# NEW
    ├── integration/image_lifecycle_test.go   # NEW
    ├── integration/version_conflict_test.go  # NEW
    ├── integration/ordering_test.go          # NEW: an edit does not move a collectible
    └── contract/edit_delete_test.go          # NEW

frontend/
├── app/collection/[id]/edit/
│   ├── page.tsx                  # NEW: RSC, forwards the cookie, 404 → not-found
│   ├── loading.tsx               # NEW
│   └── not-found.tsx             # NEW
├── components/collection/
│   ├── CollectibleForm.tsx       # NEW: shared by add and edit
│   ├── AddCollectibleForm.tsx    # CHANGED: thin wrapper over CollectibleForm
│   ├── EditCollectibleForm.tsx   # NEW: wrapper carrying version and delete
│   ├── DeleteCollectibleDialog.tsx # NEW: native <dialog>
│   ├── CollectibleCard.tsx       # CHANGED: an edit affordance carrying the active filter
│   └── VersionConflictNotice.tsx # NEW: the designed state for a refused save
├── lib/
│   ├── api/collectibles.ts       # CHANGED: get, edit, delete
│   ├── api/errors.ts             # CHANGED: recognise version_conflict
│   └── types/api.ts              # REGENERATED
└── tests/
    ├── unit/collectible-form.test.tsx          # NEW
    ├── unit/delete-dialog.test.tsx             # NEW
    └── e2e/{edit-collectible,delete-collectible,edit-photograph}.spec.ts  # NEW
```

**Structure Decision**: the existing web-application split is unchanged — `backend/` owns domain,
validation, authorization and persistence; `frontend/` renders. This feature adds no new layer and
no new package boundary. The one new backend package member worth noting is
`collection/image_cleanup.go`, which is a service concern rather than a transport or store one, and
sits with the use cases accordingly.

## Phase sequencing

The work has one hard ordering constraint and is otherwise parallel:

1. **Contract and migrations first.** The OpenAPI document and both migrations land before handlers,
   because the frontend's types are generated from the former and every store change depends on the
   latter.
2. **Domain, then store, then service, then transport.** Each layer depends only on the one below it.
3. **Frontend after the contract**, but not after the backend: generated types are enough to build
   against, and the e2e suite is what joins them.
4. **The form generalisation before the edit screen**, so the edit screen is written against the
   shared component rather than being retrofitted onto it.

## Risks this plan is deliberately managing

| Risk | How the design handles it |
|---|---|
| A `PUT` that omits `imageId` silently deletes a collector's photograph | Stated in the contract's field description, and asserted by an integration test that edits only the name and checks the image survives (research.md Decision 3) |
| 404 and 409 collapse into "zero rows affected" | `SELECT … FOR UPDATE` before the update, so each case is distinguishable (research.md Decision 6) |
| A photograph stays fetchable after its collectible is deleted | The image row is deleted in the same transaction; files are hygiene, not the guarantee (research.md Decision 4) |
| A storage fault makes a collectible undeletable | File removal is outside the transaction and queued; the collector's operation never waits on it (research.md Decision 5) |
| Add and edit validation drift apart | One shared validator, one rule set; the unit suite exercises both paths against it (research.md Decision 10) |
| Editing quietly reorders the gallery | `created_at` is never written, and an integration test asserts position is unchanged (research.md Decision 11) |
| A reintroduced `router.refresh()` revives the navigation race | A single `router.push` after save and delete, with the reason recorded where the next person will read it (research.md Decision 14) |

## Post-design constitution re-check

Re-evaluated after the contract, data model and quickstart were written. No principle is violated and
nothing needed justification, so there is no Complexity Tracking table. Two points are worth
recording because they changed during design:

- **The photograph guarantee moved layers.** It was originally reasoned about as a consequence of
  removing the reference. Reading `store/postgres/images.go` showed that renditions are authorised on
  the image's own `collector_id`, so the guarantee had to become an in-transaction row deletion. The
  requirement did not change; the mechanism did, and Principle IV is satisfied by construction
  instead of by assumption.
- **The contract gained a breaking change.** `Collectible.version` is required, so every consumer
  sees a new field. There is one consumer, it is generated, and Principle III asks that breaking
  changes be intentional and documented rather than avoided — hence 0.2.0 and the note in
  `contracts/README.md`.
