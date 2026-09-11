-- One uploaded primary image and its derived gallery rendition. A row exists from the moment of
-- upload, before any collectible references it, because uploading is a separate operation
-- (research.md Decision 3).
CREATE TABLE collectible_images (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    collector_id  uuid NOT NULL REFERENCES collectors (id) ON DELETE CASCADE,
    original_key  text NOT NULL,
    rendition_key text NOT NULL,
    content_type  text NOT NULL,
    byte_size     bigint NOT NULL,
    width         integer NOT NULL,
    height        integer NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),

    -- FR-009: only these three formats, as determined by decoding rather than by what the client
    -- declared.
    CONSTRAINT collectible_images_content_type
        CHECK (content_type IN ('image/jpeg', 'image/png', 'image/webp')),

    -- FR-010: the 10 MB ceiling is enforced here as well as in the request path, so no future code
    -- path can bypass it.
    CONSTRAINT collectible_images_byte_size
        CHECK (byte_size > 0 AND byte_size <= 10485760),

    CONSTRAINT collectible_images_dimensions
        CHECK (width > 0 AND height > 0),

    -- Redundant against the primary key on its own, and deliberately so: it is the target of the
    -- composite reference from collectibles, which is what makes the database refuse a collectible
    -- that points at another collector's image (FR-015).
    CONSTRAINT collectible_images_id_collector_unique UNIQUE (id, collector_id)
);

CREATE INDEX collectible_images_collector_idx ON collectible_images (collector_id);
