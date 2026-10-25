-- The Trash (feature 010).
--
-- A deleted collectible keeps its row. deleted_at says when it was moved to the Trash; NULL means it
-- is in the vault. Everything else about the row — id, created_at, version, attributes, image — is
-- untouched, which is what makes a restore a single UPDATE and returns the collectible to its
-- original gallery position by construction (research.md Decisions 1 and 2).
--
-- Every read of the vault must now say deleted_at IS NULL. That is the cost of a column over a
-- second table, and the integration suite checks each read with something in the Trash.
ALTER TABLE collectibles ADD COLUMN deleted_at timestamptz;

-- The Trash listing: one collector's, most recently deleted first, keyset-paginated.
CREATE INDEX collectibles_trash_idx ON collectibles (collector_id, deleted_at DESC, id DESC)
    WHERE deleted_at IS NOT NULL;

-- The purge: expired items across every collector, oldest deletion first.
CREATE INDEX collectibles_trash_expiry_idx ON collectibles (deleted_at)
    WHERE deleted_at IS NOT NULL;

-- A photograph belongs to at most one collectible (FR-018).
--
-- Nothing enforced this before, and with a Trash it starts to matter: a trashed collectible's
-- photograph is still a valid image, so another collectible could name it, and destroying the
-- trashed one would delete the image row — whose ON DELETE SET NULL would silently strip the other
-- collectible's photograph.
--
-- If two collectibles already share an image (possible only by a hand-crafted request), this fails.
-- Deliberately: choosing which one loses its photograph is not a decision a migration should make.
CREATE UNIQUE INDEX collectibles_image_unique ON collectibles (image_id)
    WHERE image_id IS NOT NULL;
