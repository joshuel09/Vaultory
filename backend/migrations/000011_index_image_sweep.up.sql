-- Two indexes the abandoned-upload sweep needs (feature 009, research.md Decision 4).
--
-- The sweep takes unreferenced uploads oldest first, so it orders collectible_images by age.
CREATE INDEX collectible_images_created_idx ON collectible_images (created_at);

-- And it asks, for each candidate, whether any collectible references it. Nothing indexed that
-- question until now — which also means the foreign key's ON DELETE SET NULL, and every image row
-- feature 006 deletes, has been scanning collectibles to answer it.
CREATE INDEX collectibles_image_idx ON collectibles (image_id);
