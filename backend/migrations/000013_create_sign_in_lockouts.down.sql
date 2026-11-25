-- Every count and lock is lost, which only means addresses locked now can sign in again at once.
DROP FUNCTION IF EXISTS sign_in_clear(text);
DROP FUNCTION IF EXISTS sign_in_record_failure(text);
DROP FUNCTION IF EXISTS sign_in_is_locked(text);
DROP TABLE IF EXISTS sign_in_lockouts;
