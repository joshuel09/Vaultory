# Quickstart: verifying the Trash

Backend checks need PostgreSQL with migrations applied (`make test`, or set
`VAULTORY_TEST_DATABASE_URL`).

## A — Undo (US1)

Browser: edit a collectible with a photograph → Delete → confirm. The gallery shows "Moved to the
Trash" with Undo. Undo: the collectible is back in the same position, with its photograph.
Automated: `frontend` unit test for `UndoDeleteNotice`; integration `TestRestoreBringsBackEverything`.

## B — The Trash (US2)

Browser: delete two collectibles; open Trash from the vault header; both are listed, newest deleted
first, with photographs and "Removed for good on …". Restore one.
Automated: integration `TestTrashListing*`, `TestTrashIsInvisibleToEveryVaultView`; contract
`trash_test.go`.

## C — The window (FR-009, SC-004)

Automated: integration test backdates `deleted_at` past 30 days. The item is unlisted, its restore is
refused, and the purge destroys it and its photograph.

## D — Permanent deletion (US3, SC-005)

Browser: Trash → Delete permanently → the confirmation names it; Enter cancels; confirm destroys it.
Automated: integration `TestDeletePermanently*`: the rendition answers not-found at once and the
files are gone after the drain.

## E — The race (FR-012)

Automated: integration test races restore against purge for an item exactly at its expiry, many
times. Each ends restored-and-kept or destroyed-and-refused.

## F — Privacy (FR-013, SC-006)

Automated: another collector's restore and permanent delete answer exactly as for an absent id; their
Trash listing is empty.
