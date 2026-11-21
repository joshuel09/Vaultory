-- Sign-in is locked for an email address after five consecutive failures, for fifteen minutes
-- (feature 013, issue #99).
--
-- The rule lives here rather than in the frontend that calls it: every instance shares one count,
-- a restart does not reset it, and the increment is one atomic statement however many failures
-- arrive at once. Better Auth's own rate limiter cannot do this — it counts per network address and
-- path, which is the gap this closes.
--
-- address_key is an HMAC of the normalized address made with the server's secret, never the
-- address. It is computed for every address tried, registered or not, so a lock looks the same for
-- an address that has no account (FR-006, FR-009).
CREATE TABLE sign_in_lockouts (
    address_key  text        PRIMARY KEY,
    failures     integer     NOT NULL CHECK (failures >= 0),
    locked_until timestamptz,
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- A count nobody has touched for 30 days is forgotten; this finds those rows.
CREATE INDEX sign_in_lockouts_updated_idx ON sign_in_lockouts (updated_at);

-- True while sign-in for the address is refused.
CREATE FUNCTION sign_in_is_locked(p_key text) RETURNS boolean
LANGUAGE sql STABLE AS $$
    SELECT EXISTS (
        SELECT 1 FROM sign_in_lockouts WHERE address_key = p_key AND locked_until > now()
    );
$$;

-- One failed sign-in. During a lock nothing changes: the attempt was refused before its password
-- was checked, so it neither counts nor extends the lock. After a lock has expired the count starts
-- again at one. The fifth consecutive failure starts a fifteen-minute lock.
--
-- One INSERT … ON CONFLICT: concurrent failures for the same address queue on the row and each
-- sees the one before it, so none is lost (SC-005).
CREATE FUNCTION sign_in_record_failure(p_key text) RETURNS void
LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO sign_in_lockouts AS l (address_key, failures, locked_until, updated_at)
    VALUES (p_key, 1, NULL, now())
    ON CONFLICT (address_key) DO UPDATE SET
        failures = CASE
            WHEN l.locked_until > now() THEN l.failures
            WHEN l.locked_until IS NOT NULL THEN 1
            ELSE l.failures + 1
        END,
        locked_until = CASE
            WHEN l.locked_until > now() THEN l.locked_until
            WHEN l.locked_until IS NULL AND l.failures + 1 >= 5 THEN now() + interval '15 minutes'
            ELSE NULL
        END,
        updated_at = now();

    -- Housekeeping, bounded: an address that failed once and never came back would otherwise keep
    -- a row forever.
    DELETE FROM sign_in_lockouts
    WHERE address_key IN (
        SELECT address_key FROM sign_in_lockouts
        WHERE updated_at < now() - interval '30 days'
        ORDER BY updated_at
        LIMIT 100
    );
END;
$$;

-- A successful sign-in, or a completed password reset: the count and any lock are gone.
CREATE FUNCTION sign_in_clear(p_key text) RETURNS void
LANGUAGE sql AS $$
    DELETE FROM sign_in_lockouts WHERE address_key = p_key;
$$;

-- The frontend's restricted role gets this table and these three functions, and nothing else new.
-- PostgreSQL lets every role execute a new function unless told otherwise.
REVOKE ALL ON FUNCTION sign_in_is_locked(text), sign_in_record_failure(text), sign_in_clear(text)
    FROM PUBLIC;
GRANT EXECUTE ON FUNCTION sign_in_is_locked(text), sign_in_record_failure(text), sign_in_clear(text)
    TO vaultory_auth;
GRANT SELECT, INSERT, UPDATE, DELETE ON sign_in_lockouts TO vaultory_auth;
