-- One entry in one collector's vault. Each added entry is a distinct row; two identical entries
-- remain two rows and are never merged or collapsed into a quantity (FR-023, FR-024).
CREATE TABLE collectibles (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    collector_id      uuid NOT NULL REFERENCES collectors (id) ON DELETE CASCADE,
    name              text NOT NULL,
    collection_status text NOT NULL,
    "character"       text,
    series            text,
    manufacturer      text,
    category          text,
    scale             text,
    edition           text,
    purchase_price    numeric(12, 2),
    purchase_date     date,
    release_date      date,
    notes             text,
    image_id          uuid,
    created_at        timestamptz NOT NULL DEFAULT now(),

    -- FR-002, FR-003: the check is on the trimmed length, so a whitespace-only name fails at the
    -- database as well as in the domain.
    CONSTRAINT collectibles_name_length
        CHECK (char_length(btrim(name)) BETWEEN 1 AND 200),

    -- FR-004: exactly four statuses, closed by the database as well as the domain.
    CONSTRAINT collectibles_collection_status
        CHECK (collection_status IN ('owned', 'preordered', 'wishlist', 'sold')),

    CONSTRAINT collectibles_character_length    CHECK ("character" IS NULL OR char_length("character") <= 200),
    CONSTRAINT collectibles_series_length       CHECK (series IS NULL OR char_length(series) <= 200),
    CONSTRAINT collectibles_manufacturer_length CHECK (manufacturer IS NULL OR char_length(manufacturer) <= 200),
    CONSTRAINT collectibles_category_length     CHECK (category IS NULL OR char_length(category) <= 100),
    CONSTRAINT collectibles_scale_length        CHECK (scale IS NULL OR char_length(scale) <= 50),
    CONSTRAINT collectibles_edition_length      CHECK (edition IS NULL OR char_length(edition) <= 200),
    CONSTRAINT collectibles_notes_length        CHECK (notes IS NULL OR char_length(notes) <= 2000),

    -- FR-017: zero is a recorded amount (a gift); negative never is.
    CONSTRAINT collectibles_purchase_price_non_negative
        CHECK (purchase_price IS NULL OR purchase_price >= 0),

    -- FR-018: a coarse backstop only. The domain performs this check against the collector's own
    -- current date; this constraint is evaluated in the database session's timezone and is
    -- deliberately the looser of the two.
    CONSTRAINT collectibles_purchase_date_not_future
        CHECK (purchase_date IS NULL OR purchase_date <= CURRENT_DATE),

    -- FR-015: composite, not single-column. A single-column key would let a collectible reference
    -- an image owned by a different collector, leaving privacy defended only by application code.
    CONSTRAINT collectibles_image_same_owner
        FOREIGN KEY (image_id, collector_id)
        REFERENCES collectible_images (id, collector_id)
        ON DELETE SET NULL
);

-- release_date is deliberately unconstrained: past or future, in any combination with status and
-- purchase date, is legitimate (FR-019).
