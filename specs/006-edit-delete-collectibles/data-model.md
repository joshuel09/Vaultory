# Data Model: Edit & Delete Collectibles

**Feature**: `specs/006-edit-delete-collectibles` | **Date**: 2026-10-01

This feature changes two things about the schema and one thing about how the existing tables are
written. Everything else in `specs/001-add-browse-collectibles/data-model.md` still holds.

---

## Migration 000009 — a version on every collectible

```sql
ALTER TABLE collectibles
    ADD COLUMN version integer NOT NULL DEFAULT 1,
    ADD CONSTRAINT collectibles_version_positive CHECK (version >= 1);
```

`version` starts at 1 for every existing row and rises by one on each successful edit (FR-027a). It
is the value an edit is checked against; the check is a plain equality, which is why this is an
integer and not a timestamp (research.md Decision 1).

**Down**: `ALTER TABLE collectibles DROP CONSTRAINT collectibles_version_positive, DROP COLUMN
version;` — reversible with no data loss beyond the counter itself, which has no meaning outside
this mechanism.

**Not added**: `updated_at`. Nothing displays or orders by it, and FR-028 requires that editing not
affect ordering. See research.md Decision 1.

---

## Migration 000010 — files whose rows are already gone

```sql
CREATE TABLE pending_image_deletions (
    image_id      uuid PRIMARY KEY,
    collector_id  uuid NOT NULL,
    original_key  text NOT NULL,
    rendition_key text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX pending_image_deletions_created_idx ON pending_image_deletions (created_at);
```

A row here means: *the `collectible_images` row is deleted, so nobody can fetch this image; these two
files are still on disk and should not be.* It is written in the same transaction that deletes the
image row, so the two can never disagree (FR-020a, research.md Decision 5).

**No foreign keys, deliberately.** `image_id` cannot reference `collectible_images` because that row
is gone by the time this one exists — that is the entire point. `collector_id` is recorded for
diagnosis, not for joining, and must not cascade: a collector's deletion is precisely when their
files most need removing.

**Down**: `DROP TABLE pending_image_deletions;` — any queued rows are lost, which leaves files on
disk that nothing references. Unreachable through the API either way, so the reversal is safe; the
down migration's comment says so rather than leaving someone to work it out.

---

## `collectibles` — what an edit writes

| Column | Edit behaviour |
|---|---|
| `id`, `collector_id`, `created_at` | Never written by an edit. `created_at` is what FR-028 depends on. |
| `version` | `version + 1`, in the same statement as the change. |
| `name`, `collection_status` | Replaced. Required, validated identically to adding (FR-010, FR-011). |
| `character`, `series`, `manufacturer`, `category`, `scale`, `edition`, `notes` | Replaced, including with `NULL` when the collector cleared the field (FR-005). |
| `purchase_price` | Replaced as `numeric(12,2)`, bound as a decimal string and cast in SQL, never through a float (FR-013). |
| `purchase_date`, `release_date` | Replaced, including with `NULL`. |
| `image_id` | Replaced. Absent or null in the request means no photograph (research.md Decision 3). |

Every statement carries `collector_id` as a predicate (FR-030). There is no edit or delete path that
locates a row by `id` alone.

### State transitions

`collection_status` moves freely between `owned`, `preordered`, `wishlist` and `sold`, in any
direction, with no transition forbidden (FR-006). There is no state machine; the four values are a
closed set enforced by `collectibles_collection_status`, and that is the whole rule.

A collectible has one lifecycle change this feature introduces: it can cease to exist. There is no
intermediate deleted state, no tombstone row, and no restore (Out of Scope).

---

## The three write transactions

### Edit

1. `SELECT … FROM collectibles WHERE id = $1 AND collector_id = $2 FOR UPDATE`
   — no row → **404** (FR-029, FR-031).
2. `version != expectedVersion` → **409**, carrying the collectible as it now is (FR-027).
3. If the request's `imageId` differs from the stored one and is not null, confirm it belongs to this
   collector; the composite foreign key is the real guarantee, this produces a field-level message
   (FR-021).
4. `UPDATE collectibles SET …, version = version + 1 WHERE id = $1 AND collector_id = $2`.
5. If the stored `image_id` was replaced or removed: `DELETE FROM collectible_images WHERE id = $old
   AND collector_id = $2`, and `INSERT INTO pending_image_deletions …` with its storage keys.
6. Commit. The whole edit lands or none of it does (FR-009).

### Delete

1. `DELETE FROM collectibles WHERE id = $1 AND collector_id = $2 RETURNING image_id`
   — no row → nothing happened; still **204** (FR-025).
2. If an `image_id` came back: delete that `collectible_images` row and queue its files.
3. Commit, then log the deletion event — identifiers only, and only because a row was actually
   deleted (FR-040, research.md Decision 8).

`collectible_submissions` rows cascade on `collectible_id`, so a collectible added within the
idempotency window deletes cleanly. The consequence is recorded as an accepted edge case in the
spec: a retried add whose collectible was deleted creates a new one.

### Drain

Outside any collector-facing transaction: read a bounded batch from `pending_image_deletions`, delete
each file from the image store, delete the row. A file that is already absent counts as deleted. A
failure leaves the row for the next drain (FR-020a).

---

## Entities as the domain sees them

- **`collectible.Collectible`** gains `Version int`.
- **`collectible.EditDraft`** is new: the same submitted fields as `Draft`, with `ExpectedVersion`
  in place of `SubmissionKey`. Both validate through one shared routine and produce one
  `Validated` (research.md Decision 10).
- **`postgres.Row`** gains the version by way of `Collectible`, so the gallery, the add response and
  the single-collectible response all carry it from one projection.
- **`postgres.PendingImageDeletion`** is new and lives only between the store and the drain.

## What the API exposes

`version` appears on every collectible the API returns, including in the gallery — one projection,
one shape, no separate "detail" representation to drift from the list one. `expectedVersion` is
required on an edit and appears nowhere else.
