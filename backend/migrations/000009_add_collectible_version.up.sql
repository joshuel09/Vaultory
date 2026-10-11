-- A marker that changes whenever a collectible changes, so a save made from a stale view can be
-- refused rather than applied over someone's newer edit (FR-027, FR-027a).
--
-- An integer rather than a timestamp: the marker's only job is equality, and two edits inside the
-- same clock tick must not look identical (research.md Decision 1). Existing rows start at 1.
ALTER TABLE collectibles
    ADD COLUMN version integer NOT NULL DEFAULT 1;

-- The counter only ever rises. A row at version 0 would mean something has written it by hand.
ALTER TABLE collectibles
    ADD CONSTRAINT collectibles_version_positive CHECK (version >= 1);
