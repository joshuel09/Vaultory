# Research: Sweep Abandoned Uploads

## 1. Grace period: 30 days

- **Decision**: 30 days, one named constant.
- **Rationale**: That is how long a session survives unused. An upload older than that belongs, in
  the normal case, to a form that has outlived the session it was made in. It is *not* a hard bound:
  signing in again in another tab revives an old form, so the guarantee rests on Decision 2 and on
  the clear refusal in FR-007, not on the number.
- **Alternatives considered**: 24 hours (a form left open over a weekend would lose its
  photograph); 90 days (the absolute session cap; triples storage held for nobody, for no gain in
  safety, since no number is a guarantee); configurable (YAGNI; nothing needs to vary it).

## 2. The race with a save is settled by row locks

- **Decision**: one transaction. First
  `SELECT … FROM collectible_images WHERE created_at < cutoff AND NOT EXISTS (reference) ORDER BY created_at LIMIT n FOR UPDATE SKIP LOCKED`,
  then a second statement deletes those ids *again conditioned on* `NOT EXISTS (reference)`, returning
  the keys into `pending_image_deletions`.
- **Rationale**: a save that references an upload makes the foreign key take `FOR KEY SHARE` on the
  upload's row, and `FOR UPDATE` conflicts with that. So there are three cases:
  - **The save is in flight.** It holds the key-share lock, so `SKIP LOCKED` passes over the row and
    the upload survives this sweep.
  - **The save committed after the sweep's first snapshot.** The second statement has a fresh
    snapshot, sees the reference and leaves the row alone.
  - **The save starts after the sweep has locked the row.** Its foreign-key check waits for the
    sweep to commit, then fails because the row is gone. That surfaces as the existing unknown-image
    violation, which is FR-007.

  The only outcomes are the two FR-006 permits, and the sweep never waits on a save. The foreign key's `ON DELETE SET NULL` can never fire
  from the sweep, because nothing references a row at the moment it is deleted.
- **Alternatives considered**: one `DELETE … WHERE NOT EXISTS` statement. Planned as unsafe, on the
  reasoning that READ COMMITTED might not re-check the subquery after waiting for the row's lock,
  letting `ON DELETE SET NULL` strip a saved collectible's photograph. **Tested, that did not
  happen**: swapped in for the real implementation, it blocked on the open save, then kept the upload
  once the save committed. What it does do is *wait*, and the sweep runs inside an upload request, so
  one collector's upload would stall behind another's save (SC-005). That, plus not resting
  correctness on subquery re-evaluation after a lock wait, is why the locking version stays. The
  deterministic test asserts both, and the swapped-in version fails it on the wait. An advisory lock
  per upload would also work, but adds a second locking convention where row locks already suffice.

## 3. Triggered by uploads and at start-up, no ticker

- **Decision**: run one bounded sweep (20 rows) after each successful upload, then drain; and once
  at start-up, before the existing drain.
- **Rationale**: abandoned uploads only accrue as fast as uploads happen, so sweeping on upload works
  the backlog exactly as it grows. This is the same reasoning as feature 006 Decision 5, which
  rejected a ticker as lifecycle machinery for non-urgent work. The upload's own success never
  depends on the sweep (FR-010).
- **Alternatives considered**: a ticker (rejected as in 006); sweeping on add or edit (those reduce
  orphans, not create them).

## 4. Two indexes

- **Decision**: migration 000011 adds `collectible_images (created_at)` and `collectibles (image_id)`,
  with a down migration that drops both.
- **Rationale**: the sweep orders by age and anti-joins on `collectibles.image_id`, and there was no
  index on either. `collectibles.image_id` also serves the foreign key's existing `ON DELETE SET NULL`
  and feature 006's deletes, which until now scanned `collectibles`.
