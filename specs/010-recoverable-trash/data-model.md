# Data Model: Undo and a Recoverable Trash

## `collectibles` (changed)

| Column | Type | Change |
|---|---|---|
| `deleted_at` | `timestamptz NULL` | **new**. NULL: in the vault. Set: moved to the Trash at that moment |

Nothing else changes, and that is the point: `id`, `created_at`, `version`, the attributes and
`image_id` are untouched by trashing and restoring.

**State transitions**

```text
in vault ──delete──▶ in Trash ──restore (≤30 days)──▶ in vault
                        │
                        ├──delete permanently──▶ destroyed (row + image row; files queued)
                        └──30 days pass, purge──▶ destroyed (same)
```

Deleting something already in the Trash leaves `deleted_at` unchanged (FR-004).

**Indexes** (migration 000012)

- `collectibles_trash_idx ON collectibles (collector_id, deleted_at DESC, id DESC) WHERE deleted_at IS NOT NULL`: the Trash listing, keyset-paginated.
- `collectibles_trash_expiry_idx ON collectibles (deleted_at) WHERE deleted_at IS NOT NULL`: the purge, across collectors.

**Down migration**: queues the files of trashed collectibles' images, deletes those image rows and
the trashed collectibles, then drops the column. Reversing the feature destroys the Trash the way 006
destroyed deletions; it never brings deleted collectibles back into vaults.

## `collectible_submissions` (behaviour only)

A collectible's submission row is deleted when it is moved to the Trash (research Decision 5).

## `collectible_images`, `pending_image_deletions`

Unchanged. A trashed collectible keeps its image row; destruction uses 006's `releaseImage`.
