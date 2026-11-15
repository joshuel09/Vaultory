-- Two indexes the abandoned-upload sweep needs (feature 009, research.md Decision 4).
--
-- The sweep takes unreferenced uploads oldest first, and this index was added to order it. Measured
-- at 200,000 images the planner does not use it for that — it takes a hash anti join over both
-- tables and sorts what survives — so it is not what keeps the sweep cheap; nothing does (#72).
CREATE INDEX collectible_images_created_idx ON collectible_images (created_at);

-- And it asks, for each candidate, whether any collectible references it. Nothing indexed that
-- question until now — which also means the foreign key's ON DELETE SET NULL, and every image row
-- feature 006 deletes, has been scanning collectibles to answer it.
CREATE INDEX collectibles_image_idx ON collectibles (image_id);
