-- Any queued rows are lost, which leaves their files in the image store with nothing referencing
-- them. That is safe rather than merely acceptable: the collectible_images rows are already gone,
-- so the files are unreachable through the API whether or not this table exists. Reversing this
-- migration forfeits the cleanup, not the guarantee.
DROP TABLE IF EXISTS pending_image_deletions;
