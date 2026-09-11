-- Keyset pagination in creation order, newest first, per collector (FR-034, FR-035).
-- The id tiebreaker makes the order total, so entries created in the same instant cannot be
-- skipped or repeated across pages (research.md Decision 7).
CREATE INDEX collectibles_gallery_idx
    ON collectibles (collector_id, created_at DESC, id DESC);

-- The same page with a status filter applied (FR-037).
CREATE INDEX collectibles_status_gallery_idx
    ON collectibles (collector_id, collection_status, created_at DESC, id DESC);

-- There is intentionally no unique constraint across name and attributes: duplicates are
-- legitimate and required (FR-023).
