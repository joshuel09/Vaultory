-- Reversible with no collection data lost. The counter has no meaning outside the stale-save
-- check, so dropping it discards nothing a collector recorded.
ALTER TABLE collectibles DROP CONSTRAINT IF EXISTS collectibles_version_positive;
ALTER TABLE collectibles DROP COLUMN IF EXISTS version;
