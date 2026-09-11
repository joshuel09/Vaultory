-- Collectors own vaults. Present as a real table with a real key from the start, even though
-- authentication is out of scope, so ownership is enforceable now and nothing needs restructuring
-- when registration and login arrive (research.md Decision 1).
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE collectors (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name  text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT collectors_display_name_length
        CHECK (char_length(btrim(display_name)) BETWEEN 1 AND 100)
);
