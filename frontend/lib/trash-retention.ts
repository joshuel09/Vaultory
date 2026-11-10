/**
 * How long a deleted collectible is kept in the Trash, in days, for the sentences that say so.
 *
 * The server owns the rule: `trashRetention` in backend/internal/collection/trash.go decides when
 * something is destroyed, and every `purgeAt` the API returns comes from it. This is only the
 * number those sentences state, kept in one place, and a backend unit test fails when the two
 * disagree, so changing the rule cannot leave the copy saying something else (feature 010, #73).
 */
export const TRASH_RETENTION_DAYS = 30
