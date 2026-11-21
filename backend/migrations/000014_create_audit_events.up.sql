-- An audit log of account and collection changes (feature 014, issue #100).
--
-- Both services write here: the Go service the collection changes, inside the same transaction as
-- the change; the frontend the account events, from Better Auth's hooks. Nothing personal is kept —
-- no email, no name, no network address, no password or token, nothing of a collectible but its id.
-- A failed sign-in carries feature 013's one-way address key instead of the address.
CREATE TABLE audit_events (
    id          bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    action      text        NOT NULL CHECK (action IN (
                    'account.registered', 'account.signed_in', 'account.sign_in_failed',
                    'account.sign_in_refused', 'account.signed_out', 'account.email_verified',
                    'account.password_reset',
                    'collectible.created', 'collectible.edited', 'collectible.deleted')),
    outcome     text        NOT NULL CHECK (outcome IN ('succeeded', 'failed', 'refused')),
    account_id  text,
    subject_id  text
);

-- Retention finds expired rows by age, and reads only those, however large the table grows.
CREATE INDEX audit_events_occurred_idx ON audit_events (occurred_at);

-- Append-only, enforced here rather than trusted to either service (FR-006). The Go service
-- connects as the table's owner, so privileges alone cannot stop it; triggers apply to the owner.
CREATE FUNCTION audit_events_refuse_change() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' AND OLD.occurred_at < now() - interval '1 year' THEN
        RETURN OLD;  -- retention: an entry over a year old may go
    END IF;
    RAISE EXCEPTION 'audit entries cannot be % (feature 014, FR-006)', lower(TG_OP)
        USING ERRCODE = 'insufficient_privilege';
END;
$$;

CREATE TRIGGER audit_events_no_update BEFORE UPDATE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION audit_events_refuse_change();
CREATE TRIGGER audit_events_retention BEFORE DELETE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION audit_events_refuse_change();
CREATE TRIGGER audit_events_no_truncate BEFORE TRUNCATE ON audit_events
    FOR EACH STATEMENT EXECUTE FUNCTION audit_events_refuse_change();

-- Removes up to p_limit entries over a year old, oldest first; returns how many (FR-007). Called by
-- the Go service at start-up and after deletions — there is no schedule.
CREATE FUNCTION audit_purge_expired(p_limit integer) RETURNS integer
LANGUAGE plpgsql AS $$
DECLARE
    removed integer;
BEGIN
    DELETE FROM audit_events
    WHERE id IN (
        SELECT id FROM audit_events
        WHERE occurred_at < now() - interval '1 year'
        ORDER BY occurred_at
        LIMIT p_limit
    );
    GET DIAGNOSTICS removed = ROW_COUNT;
    RETURN removed;
END;
$$;
REVOKE ALL ON FUNCTION audit_purge_expired(integer) FROM PUBLIC;

-- The frontend's restricted role may add entries and do nothing else with the log (FR-008).
GRANT INSERT ON audit_events TO vaultory_auth;
