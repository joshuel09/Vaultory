-- Submission keys, so a retried add is answered with the collectible it already created rather
-- than a second one (FR-047). Deliberate duplicates are unaffected: they arrive with a different
-- key, and take the normal path (FR-023).
CREATE TABLE collectible_submissions (
    collector_id   uuid NOT NULL REFERENCES collectors (id) ON DELETE CASCADE,
    submission_key text NOT NULL,
    collectible_id uuid NOT NULL REFERENCES collectibles (id) ON DELETE CASCADE,
    created_at     timestamptz NOT NULL DEFAULT now(),

    -- This constraint, not an application-level check, is what holds under a genuine concurrent
    -- double submit: two simultaneous requests cannot both pass a check-then-insert, but they
    -- cannot both insert this key either.
    PRIMARY KEY (collector_id, submission_key),

    CONSTRAINT collectible_submissions_key_length
        CHECK (char_length(submission_key) BETWEEN 1 AND 200)
);

CREATE INDEX collectible_submissions_created_idx ON collectible_submissions (created_at);
