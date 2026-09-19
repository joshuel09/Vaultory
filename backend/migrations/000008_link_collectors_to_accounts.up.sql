-- Attach accounts to collectors, and restrict what the frontend's database role can reach.
--
-- Order matters here and is deliberate: remove the accountless fixtures first, then add the
-- column, then constrain it. `NOT NULL` is only possible once no collector without an account
-- remains.

-- 1. Remove the two seeded development collectors from migration 000005 (FR-017a).
--
-- They are fixtures, not people: every collectible they own came from a test run. Deleting them is
-- what makes "no vault is orphaned" trivially true, and it leaves no vault reachable without an
-- account. The cascades on collectibles, images, and submissions do the rest.
DELETE FROM collectors
 WHERE id IN (
     '11111111-1111-4111-8111-111111111111',
     '22222222-2222-4222-8222-222222222222'
 );

-- 2. One account, one collector — enforced by the schema rather than by application code (FR-016).
ALTER TABLE collectors
    ADD COLUMN user_id text UNIQUE REFERENCES "user" ("id") ON DELETE CASCADE;

ALTER TABLE collectors
    ALTER COLUMN user_id SET NOT NULL;

-- 3. A collector appears when an account does.
--
-- Registration happens inside Better Auth, in Next.js. Having Next.js also write to `collectors`
-- would put persistence of a backend-owned table in the presentation layer, which the constitution
-- forbids. A trigger keeps the rule in the database and makes it atomic with the insert that
-- causes it, so an account without a vault cannot exist even if registration fails partway
-- (research Decision 3).
--
-- SECURITY DEFINER because the Better Auth role has no privileges on `collectors`: the privilege
-- belongs to the trigger, not to whoever fired it. The explicit search_path is not optional — a
-- SECURITY DEFINER function without one can be redirected at attacker-controlled objects.
CREATE FUNCTION create_collector_for_user() RETURNS trigger
    LANGUAGE plpgsql
    SECURITY DEFINER
    SET search_path = public, pg_temp
AS $$
BEGIN
    INSERT INTO collectors (id, display_name, user_id)
    VALUES (gen_random_uuid(), split_part(NEW."email", '@', 1), NEW."id");
    RETURN NEW;
END;
$$;

CREATE TRIGGER user_creates_collector
    AFTER INSERT ON "user"
    FOR EACH ROW
EXECUTE FUNCTION create_collector_for_user();

-- 4. The role the frontend connects as.
--
-- Created with LOGIN and NO PASSWORD on purpose. A credential inside a tracked migration is what
-- the constitution forbids in as many words, and this repository is public; the password is set
-- afterwards from an environment variable. A LOGIN role with no password cannot authenticate, so
-- skipping that step fails closed — the frontend cannot connect — rather than leaving an open
-- account.
--
-- Roles are cluster-scoped while everything else here is database-scoped, hence the guard.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'vaultory_auth') THEN
        CREATE ROLE vaultory_auth LOGIN;
    END IF;
END
$$;

-- The database name differs between development, test, and production, and GRANT needs an
-- identifier rather than an expression.
DO $$
BEGIN
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO vaultory_auth', current_database());
END
$$;
GRANT USAGE ON SCHEMA public TO vaultory_auth;

GRANT SELECT, INSERT, UPDATE, DELETE
    ON "user", "session", "account", "verification", "rateLimit"
    TO vaultory_auth;

-- Deliberately absent: any privilege on collectors, collectibles, collectible_images, or
-- collectible_submissions. This is what keeps the claim true that a bug in the frontend cannot
-- expose another collector's vault, now that the frontend holds a database connection. PostgreSQL
-- enforces it; no application code is trusted to.
