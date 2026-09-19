-- Reverse of 000008. Reversibility is a constitution requirement, and a down migration nobody has
-- run is a guess — quickstart walkthrough F exercises this one.

DROP TRIGGER IF EXISTS user_creates_collector ON "user";
DROP FUNCTION IF EXISTS create_collector_for_user();

REVOKE ALL ON "user", "session", "account", "verification", "rateLimit" FROM vaultory_auth;
REVOKE ALL ON SCHEMA public FROM vaultory_auth;
DO $$
BEGIN
    EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM vaultory_auth', current_database());
END
$$;
DROP ROLE IF EXISTS vaultory_auth;

ALTER TABLE collectors DROP COLUMN IF EXISTS user_id;

-- Re-seed the two development fixtures removed by the up migration, reproducing migration 000005
-- exactly — same identifiers, same names, same vaultory.dev_seed guard. A down migration that
-- restores something subtly different is not a reversal.
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
