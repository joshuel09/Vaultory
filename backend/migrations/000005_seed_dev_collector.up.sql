-- Development-only collectors, so ownership and cross-collector isolation are exercisable locally
-- and by the integration suite. Guarded: a no-op unless vaultory.dev_seed is set to 'on'.
--
--   psql -c "SET vaultory.dev_seed = 'on'" ...   or   migrate with the dev database URL
--
-- Two collectors, not one, because a single collector cannot demonstrate FR-026 or FR-027.
DO $$
BEGIN
    IF current_setting('vaultory.dev_seed', true) IS DISTINCT FROM 'off' THEN
        INSERT INTO collectors (id, display_name)
        VALUES
            ('11111111-1111-4111-8111-111111111111', 'Development Collector'),
            ('22222222-2222-4222-8222-222222222222', 'Second Development Collector')
        ON CONFLICT (id) DO NOTHING;
    END IF;
END $$;
