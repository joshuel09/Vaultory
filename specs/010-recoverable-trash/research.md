# Research: Undo and a Recoverable Trash

## 1. A `deleted_at` column, not a separate table

- **Decision**: `collectibles.deleted_at timestamptz NULL`. NULL means in the vault; set means in the
  Trash since then.
- **Rationale**: The row, its id, its `created_at`, its version, its submission history and its image
  reference all survive unchanged, so a restore is a single `UPDATE` and the gallery position is
  preserved by construction. The composite foreign key to the image keeps protecting ownership
  (006 FR-021).
- **Risk, and its control**: every vault read must now say `deleted_at IS NULL`, and a query that
  forgets will show trashed items. There are six such reads: get, edit lock, list, list count,
  submission replay, and the two statistics queries. An integration test exercises each one with a
  trashed collectible present (SC-003), so a seventh read added later without the predicate fails a
  test rather than reaching a collector.
- **Alternatives considered**: moving rows to a `trashed_collectibles` table (a restore would re-insert,
  breaking the id that open edit forms and links hold, and duplicating every constraint); a view over
  live rows (it would hide the predicate in a place a reader of the query does not see).

## 2. The photograph stays while the collectible is in the Trash

- **Decision**: trashing does not touch `collectible_images`. Leaving the Trash for good — by expiry
  or permanent deletion — runs 006's `releaseImage` in the same transaction as the row deletion.
- **Rationale**: the issue requires the photograph to come back. 006 Decision 4 is unchanged in
  substance: deleting the image row is still the privacy guarantee, and it still happens
  transactionally with the collectible leaving for good. The owner can still fetch the rendition
  while it is in the Trash, which the Trash page needs (FR-014). Others never could (`GetImage` is
  collector-scoped).
- **Interaction with feature 009**: its sweep removes only *unreferenced* uploads; a trashed
  collectible still references its image, so it is left alone. No change to 009 is needed.

## 3. 30 days, and the same cutoff everywhere

- **Decision**: one constant, `trashRetention = 30 * 24h`. Restore and the Trash listing both require
  `deleted_at > now() - retention`. The purge takes `deleted_at <= now() - retention`.
- **Rationale**: one boundary means an item is never listed but unrestorable, or restorable but
  unlisted. An expired item that has not yet been purged is invisible and unrestorable; the purge
  only catches up physically.

## 4. Purge on deletion and at start-up, bounded, with row locks

- **Decision**: `PurgeExpiredTrash` locks up to 20 expired rows `FOR UPDATE SKIP LOCKED`, oldest
  deletion first, deletes each and releases its image, in one transaction. It runs after each
  deletion (then the drain) and once at start-up (before the drain). There is no ticker.
- **Rationale**: as in 006 Decision 5 and 009 Decision 3, this is non-urgent work that grows only when
  deletions happen. On the race with a restore (FR-012):
  - **The restore commits first.** It changed the row, so the purge's `FOR UPDATE` re-checks the
    qualification against the new row version, in which `deleted_at` is NULL, and excludes it.
  - **The purge locks first.** The restore's `UPDATE` waits, then finds the row gone and reports
    not-found.

  Either way, exactly one outcome. `SKIP LOCKED` keeps a deletion request from waiting on someone
  else's restore.

## 5. Trashing clears the add's idempotency record

- **Decision**: moving a collectible to the Trash deletes its `collectible_submissions` row in the
  same transaction.
- **Rationale**: otherwise a retried add (same key, inside the 24-hour window) would find the trashed
  collectible and either return it into the gallery or fail. 006 already accepts that a retried add
  after a deletion creates a new collectible. The record exists only for double-submit protection,
  which has no meaning once the collector has deleted the result.

## 6. Endpoints under `/api/trash`

- **Decision**: `GET /api/trash?cursor&limit`, `POST /api/trash/{id}/restore` (200 with the
  collectible, 404 otherwise), and `DELETE /api/trash/{id}` (204, idempotent). `DELETE
  /api/collectibles/{id}` keeps its 204 contract and now trashes.
- **Rationale**: the Trash is a different collection with different operations; keeping it out of
  `/api/collectibles` means no vault endpoint can return a trashed item by a missing parameter.
  Restore is a POST to a sub-resource because it is an action, not a field update. Restore answers
  404, not 204, for anything absent: unlike deletion, the collector must know whether it came back
  (spec edge cases).
