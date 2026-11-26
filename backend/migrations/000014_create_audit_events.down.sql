-- Reversing this discards the audit log. The triggers would refuse the deletion of recent entries
-- one by one; dropping the table removes them with it, which is what reversing a migration means.
DROP FUNCTION IF EXISTS audit_purge_expired(integer);
DROP TABLE IF EXISTS audit_events;
DROP FUNCTION IF EXISTS audit_events_refuse_change();
