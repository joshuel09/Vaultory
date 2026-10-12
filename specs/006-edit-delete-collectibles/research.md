# Phase 0 Research: Edit & Delete Collectibles

**Feature**: `specs/006-edit-delete-collectibles` | **Date**: 2026-10-01

Every decision below was taken against the code that exists, not against a general principle. Where
a decision corrects an assumption made earlier in this feature, it says so.

---

## Decision 1 — The change marker is an integer version column

**Decision**: `collectibles.version integer NOT NULL DEFAULT 1`, incremented by every successful
edit. It crosses the API as `version` on a collectible and `expectedVersion` on an edit.

**Rationale**: FR-027a needs a value that changes on every change and can be compared exactly.

- A timestamp (`updated_at`) is tempting because it is independently useful, but two edits inside
  the same clock tick are indistinguishable, and comparing timestamps across a JSON round-trip
  invites precision loss. The marker's only job is equality.
- PostgreSQL's `xmin` system column needs no migration at all, but it is an implementation detail of
  MVCC, it wraps around, and `VACUUM FREEZE` can change it without the row changing. A guarantee
  that quietly depends on vacuum behaviour is not a guarantee.
- An integer is exact, monotonic, trivially asserted in a test, and readable in a failure message.

**Alternatives considered**: `updated_at timestamptz` (rejected: equality is what is needed, and
timestamps are the wrong tool for equality); `xmin` (rejected: not a stable application value); a
hash of the row's contents (rejected: an edit that sets a field back to its previous value would be
accepted as if nothing had happened, which is a different and worse semantic).

**Note on scope**: no `updated_at` column is added. Nothing in this feature needs to display or sort
by when a collectible changed, and FR-028 explicitly requires that editing *not* affect ordering.
Adding the column "because it is usually useful" is the speculative addition Principle V forbids.

---

## Decision 2 — The version is carried in the body, not as an ETag

**Decision**: `version` is a field on the collectible response; `expectedVersion` is a required
field on the edit request. No `ETag` / `If-Match` headers.

**Rationale**: `If-Match` is the more idiomatic HTTP answer, and in a different project it would win.
Here the OpenAPI document is the source of truth and the frontend's types are generated from it
(Principle III), so a value in the schema is a value the frontend gets for free and cannot
misspell. A header is invisible to the generator, has to be plumbed by hand through the Next.js
rewrite, and would be the one part of the contract that is not type-checked.

**Alternatives considered**: `ETag` + `If-Match` with `412 Precondition Failed` (rejected: correct
but untyped at the seam, and the 412 body could not carry the current collectible as naturally).

---

## Decision 3 — Editing is a full replacement (`PUT`), and `imageId` follows that rule exactly

**Decision**: `PUT /collectibles/{collectibleId}` replaces every attribute. An `imageId` that is
absent or null means *this collectible has no photograph*.

**Rationale**: FR-010 requires that editing apply exactly the rules adding applies, with none
relaxed or added. A full replacement gets that for free: the same request shape, the same validator,
the same violations. `PATCH` would introduce a third state per field — absent meaning "leave alone"
as distinct from null meaning "clear" — and the add path has no such concept, so the two would drift
apart and SC-008 would be a matter of discipline rather than construction.

**The consequence, stated plainly because it is the easy bug in this feature**: the edit form must
send the existing `imageId` back when the collector did not touch the photograph (FR-018). If it
omits it, the photograph is removed. The contract says this in the field description, and an
integration test asserts that an edit which changes only the name keeps the image.

**Alternatives considered**: `PATCH` with per-field presence (rejected: two validation semantics for
one rule set); a separate photograph endpoint (rejected: a save that changed the name and the
photograph would then be two operations that can half-succeed, breaking FR-009).

---

## Decision 4 — Deleting the image row is the privacy guarantee; deleting the file is hygiene

**Decision**: when a photograph is replaced, removed, or its collectible deleted, the
`collectible_images` row is deleted **inside the same transaction**. The stored files are deleted
afterwards, outside the transaction, and retried if that fails.

**Rationale**: this corrects an assumption made when FR-020 was clarified. The reasoning offered
then was that removing the reference is enough because an image request is authorised against an
owning collectible. **It is not.** `Store.GetImage` (`backend/internal/store/postgres/images.go`)
authorises on `collectible_images.collector_id` alone, and it has to: the upload preview in
`ImagePicker` fetches a rendition before any collectible references it, so requiring a referencing
collectible would break adding. Unlinking a photograph therefore leaves it fetchable by its owner
forever, which is exactly what FR-020 forbids.

Deleting the row is what makes the rendition answer 404 — immediately, transactionally, and by the
same code path that already enforces ownership. The files are then unreachable through the API
whatever happens to them, which is why their removal can be retried rather than blocking the
collector.

The composite foreign key `(image_id, collector_id) REFERENCES collectible_images (id, collector_id)
ON DELETE SET NULL` means deleting the image row nulls any reference to it, so the ordering inside
the transaction is not delicate.

**Alternatives considered**: deleting the files inside the transaction (rejected: a storage fault
would make a collectible undeletable, and a transaction that spans a filesystem has no sensible
rollback); leaving the image row and relying on the reference (rejected: factually does not work,
per the above).

---

## Decision 5 — Files pending deletion are queued in a table, drained opportunistically

**Decision**: a `pending_image_deletions` table holds the storage keys of images whose rows have
been deleted. Rows are inserted in the same transaction as the deletion. After the transaction
commits, the service deletes those files and removes the queue rows. Anything left behind is retried
on the next edit or delete, and once at service start.

**Rationale**: FR-020a requires that a file which could not be destroyed is "retried rather than
abandoned", which means the intent to delete must outlive the request. A table is the smallest thing
that survives a crash. Writing it in the same transaction means the queue entry and the row deletion
cannot disagree.

Draining opportunistically rather than from a background ticker is deliberate: a ticker is a new
runtime lifecycle to start, stop, and test, for work that is never user-visible and never urgent —
the files are already unreachable. Draining on each edit or delete means the queue is worked exactly
when it grows, and the start-up sweep covers a service that is restarted while entries are pending.

**Alternatives considered**: a background ticker (rejected: lifecycle machinery for non-urgent work);
deleting files inline and logging failures (rejected: "logged" is not "retried", and FR-020a says
retried); reusing `collectible_images` as the queue by leaving unreferenced rows (rejected: it would
also sweep uploads abandoned before any save, which this feature's spec puts out of scope, and it
would blur a row that means "an image exists" with one that means "an image used to exist").

---

## Decision 6 — `SELECT … FOR UPDATE` distinguishes "not yours" from "stale"

**Decision**: an edit runs in one transaction: `SELECT … FROM collectibles WHERE id = $1 AND
collector_id = $2 FOR UPDATE`, then compare versions, then `UPDATE … SET version = version + 1`.

**Rationale**: the obvious shortcut — `UPDATE … WHERE id = $1 AND collector_id = $2 AND version =
$3` — reports zero rows affected for three different situations: no such collectible, someone
else's collectible, and a stale version. The first two must answer 404 (FR-031) and the third must
answer 409 with the current values (FR-027). Collapsing them loses the distinction the requirements
depend on.

`FOR UPDATE` also serialises two concurrent edits of the same row, so the losing one sees the
winner's version rather than both reading the same version and both succeeding. And the selected row
yields the current `image_id`, which the replace-and-remove path needs in order to know which image
row to delete.

**Alternatives considered**: conditional UPDATE with a follow-up SELECT to work out which case it was
(rejected: two round trips and a window in which the answer changes); optimistic retry (rejected:
the collector's edit is not safe to replay, which is the whole point of refusing it).

---

## Decision 7 — Deleting answers 204 whether or not anything was there

**Decision**: `DELETE /collectibles/{collectibleId}` returns `204 No Content` for an authenticated
collector in every case — deleted, already gone, never existed, belongs to someone else.

**Rationale**: FR-025 settled this. Two things fall out of it that are worth recording. First, it
satisfies FR-031 for free: an identifier belonging to another collector is answered identically to
one that never existed, and here that answer is success, which reveals nothing either way. Second,
it keeps the retry rule on the server, where Principle II requires business rules to live; the
alternative was a 404 that the browser is expected to interpret as success, which is a business rule
implemented in the frontend.

**Alternatives considered**: `404` on an absent collectible (rejected: pushes the rule into the
client); `200` with a body describing what happened (rejected: the body would differ between
"deleted" and "already gone", which is the disclosure FR-031 forbids).

---

## Decision 8 — The record of a deletion is a log event, not a table

**Decision**: a structured log event — collector id, collectible id, timestamp — emitted only when a
row was actually deleted. No database table, no API surface.

**Rationale**: FR-041 requires that the record never be surfaced in the product. A log is that by
construction: there is no query path from the application to it, so it cannot be surfaced by
accident and cannot be mistaken for the trash bin this feature deliberately does not build. A table
called `deleted_collectibles` would be one pull request away from becoming a restore feature.

It also matches what the transport already does: `requestLogger` records method, path, status and
duration and deliberately records no collector-supplied content. The deletion event carries
identifiers only — never the name, the notes, or any other attribute (FR-040).

**Emitted only on an actual deletion**: an idempotent 204 for a collectible that was already gone is
not a deletion, and logging one would make the record wrong in exactly the situation it exists to
explain.

**Alternatives considered**: an audit table (rejected: becomes a product feature under pressure, and
contradicts the out-of-scope boundary); nothing at all (rejected: an irreversible operation with no
trace leaves a collector's "where did it go" unanswerable).

---

## Decision 9 — The confirmation is a native `<dialog>`

**Decision**: the delete confirmation uses the platform `<dialog>` element opened with
`showModal()`. No dialog library is added.

**Rationale**: FR-037 wants focus moved into the dialog on open, Escape to abort, and focus restored
on close. `showModal()` provides all three, plus the inert backdrop and the top-layer stacking, with
no dependency. Principle V puts the burden of justification on the dependency, and
`@radix-ui/react-alert-dialog` would be carrying a package to re-implement what the element already
does.

The destructive button is not autofocused and is not the dialog's default submit, so Enter on an
unchanged dialog does not delete (FR-023). The Cancel button receives initial focus.

**Alternatives considered**: `@radix-ui/react-alert-dialog` (rejected: new dependency for behaviour
the platform has, and shadcn/ui is a foundation here rather than a mandate); a non-modal inline
confirmation (rejected: does not trap focus, so the keyboard requirement would be hand-rolled).

---

## Decision 10 — Add and edit share one validator

**Decision**: the per-field rules move into a shared routine. `Draft` (add) carries a
`SubmissionKey`; `EditDraft` carries an `ExpectedVersion`. Both produce the same `Validated` value
and the same `[]Violation`.

**Rationale**: SC-008 says every rule that rejects a value when adding rejects it when editing, with
none applying to only one. Sharing the code makes that true by construction. Two copies would make it
true only until the next change to either.

The two differ in exactly one field each, and those differences are real: a submission key separates
a retry from a deliberate duplicate and is meaningless for an edit, which addresses a collectible
that already exists; an expected version is meaningless for an add, which has nothing to be stale
against.

**Alternatives considered**: reusing `Draft` with an empty submission key (rejected: it would make
the key optional for adding too, dismantling FR-047 from feature 001); duplicating the rules
(rejected: SC-008 becomes a promise instead of a property).

---

## Decision 11 — Gallery position is preserved by not touching `created_at`

**Decision**: nothing. `List` orders by `(created_at DESC, id DESC)` and an edit writes neither
column.

**Rationale**: FR-028 is satisfied by construction rather than by effort, which is the best kind of
satisfied — but it is also the kind that a later "let's show recently updated first" change breaks
silently. An integration test asserts that a collectible edited after being added keeps its position
in the page, so the requirement has a failing test if anyone changes the ordering.

---

## Decision 12 — The contract stays in one document

**Decision**: the new operations are added to `specs/001-add-browse-collectibles/contracts/openapi.yaml`,
whose version goes to `0.2.0` and whose description is generalised from "feature 001" to the seam
itself. `specs/006-edit-delete-collectibles/contracts/README.md` records what this feature added and
why the document lives where it does.

**Rationale**: that file is already the single source of truth and is what `npm run generate:api`
reads. A second document would mean two sources for one seam, and either the generator reads one of
them or the script grows a merge step. Features 004 and 005 set the precedent of a `contracts/`
README pointing at where the real contract lives.

**Alternatives considered**: a per-feature contract fragment (rejected: two sources of truth);
moving the document to `backend/api/openapi.yaml` (rejected: the right destination eventually, but
it is a repository-wide move unrelated to editing and deleting — Principle V forbids folding it into
feature work).

---

## Decision 13 — The edit screen is a Server Component that forwards the cookie

**Decision**: `/collection/[id]/edit` is a Server Component that fetches the collectible from Go
directly, forwarding the request's cookies, exactly as `app/collection/page.tsx` does.

**Rationale**: server-side there is no Next.js rewrite in play, so a relative `/api/...` does not
resolve and an absolute request carries no cookie unless one is attached. The gallery already solved
this and the solution is not obvious enough to rediscover: read `VAULTORY_BACKEND_ORIGIN`, attach
`(await cookies()).toString()`, and treat 401 as "redirect to sign-in" rather than as an error state,
because a revoked cookie passes the middleware and must not produce a retry button that can never
succeed.

A 404 from that fetch renders the route's not-found state rather than the error state: the
collectible is gone or was never theirs, and a Try again button would be a lie.

---

## Decision 14 — Returning from an edit honours the filter the collector was browsing

**Decision**: the edit link on a card carries the active status filter, and the return after saving
or deleting goes back through `safeRedirect`.

**Rationale**: the spec's edge case has a collector deleting while a status filter is active and
expects the filtered view to refresh without the entry. Dropping them on an unfiltered gallery
instead loses their place. `lib/safe-redirect.ts` already exists for exactly this shape of problem —
a destination taken out of a request — and refusing to reuse it would mean a second, less careful
implementation of the same guard.

After a save or a delete: a single `router.push`, and no `router.refresh()`. The collection route is
`force-dynamic` and re-renders on navigation, so the refresh changes nothing — and it is a second
navigation that aborts the first, which is the race already documented in `AddCollectibleForm`.
