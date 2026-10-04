-- Image files whose database row is already gone.
--
-- A row here means: the collectible_images row has been deleted, so nobody can fetch this image
-- any more, but these two files are still in the image store and should not be. The row is written
-- in the same transaction that deletes the image row, so the two can never disagree (FR-020a).
--
-- Deleting the row is the privacy guarantee; deleting the files is hygiene that may be retried.
-- A rendition is authorized against collectible_images.collector_id alone — it has to be, because
-- the upload preview fetches one before any collectible references it — so unlinking an image is
-- not enough to make it unreachable, and removing the row is (research.md Decision 4).
CREATE TABLE pending_image_deletions (
    image_id      uuid PRIMARY KEY,
    collector_id  uuid NOT NULL,
    original_key  text NOT NULL,
    rendition_key text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- No foreign keys, deliberately.
--
-- image_id cannot reference collectible_images: that row is gone by the time this one exists, which
-- is the entire point. collector_id is recorded for diagnosis rather than for joining, and must not
-- cascade — a collector being deleted is precisely when their files most need removing.

-- The drain takes the oldest first, so a file that keeps failing does not starve the rest.
CREATE INDEX pending_image_deletions_created_idx ON pending_image_deletions (created_at);
