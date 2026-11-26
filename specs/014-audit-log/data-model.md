# Data Model: Audit Log

## `audit_events`

| Column | Type | Rule |
|---|---|---|
| `id` | `bigint` identity | primary key |
| `occurred_at` | `timestamptz` not null default `now()` | indexed (retention) |
| `action` | `text` not null | one of `account.registered`, `account.signed_in`, `account.sign_in_failed`, `account.sign_in_refused`, `account.signed_out`, `account.email_verified`, `account.password_reset`, `collectible.created`, `collectible.edited`, `collectible.deleted` |
| `outcome` | `text` not null | `succeeded`, `failed` or `refused` |
| `account_id` | `text` null | Better Auth user id, when known |
| `subject_id` | `text` null | a collectible id, or feature 013's address key |

No email, name, network address, password, token or collectible content.

## Enforcement

- `audit_events_no_update` — any UPDATE raises.
- `audit_events_retention` — DELETE raises unless `occurred_at < now() - interval '1 year'`.
- `audit_events_no_truncate` — TRUNCATE raises.
- `audit_purge_expired(limit) → integer` deletes up to `limit` expired rows, oldest first.

## Privileges

`vaultory_auth`: `INSERT` on `audit_events` and nothing else on it.

### Trash lifecycle integration

Collection actions are `collectible.created`, `collectible.edited`,
`collectible.trashed`, `collectible.restored`, `collectible.permanently_deleted`,
and `collectible.purged`. Each successful mutation inserts its action within the
same database transaction, including Restore and expiry purge. Refused and no-op
mutations add no success event; failed audit inserts roll back the mutation.
