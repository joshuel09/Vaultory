-- Reversing the Trash destroys what is in it, the way feature 006 destroyed deletions. It never
-- returns deleted collectibles to anybody's vault: the collector deleted them, and dropping the
-- column alone would make every one of them reappear.
--
-- Their photographs go the 006 way too: image rows deleted, files queued for the drain.

INSERT INTO pending_image_deletions (image_id, collector_id, original_key, rendition_key)
SELECT i.id, i.collector_id, i.original_key, i.rendition_key
FROM collectibles c
JOIN collectible_images i ON i.id = c.image_id AND i.collector_id = c.collector_id
WHERE c.deleted_at IS NOT NULL
ON CONFLICT (image_id) DO NOTHING;

-- Collectibles first: their composite foreign key would null image_id anyway, but deleting them
-- first keeps the image deletion below to images nothing references.
WITH trashed AS (
    DELETE FROM collectibles WHERE deleted_at IS NOT NULL RETURNING image_id
)
DELETE FROM collectible_images WHERE id IN (SELECT image_id FROM trashed WHERE image_id IS NOT NULL);

DROP INDEX IF EXISTS collectibles_image_unique;
DROP INDEX IF EXISTS collectibles_trash_expiry_idx;
DROP INDEX IF EXISTS collectibles_trash_idx;
ALTER TABLE collectibles DROP COLUMN IF EXISTS deleted_at;
