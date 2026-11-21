# Data Model: Lock Sign-In per Email Address

## `sign_in_lockouts`

| Column | Type | Rule |
|---|---|---|
| `address_key` | `text` primary key | HMAC-SHA256 of the normalized address, base64url. Never the address |
| `failures` | `integer` not null | Consecutive failures since the last success, reset or expired lock; `CHECK (failures >= 0)` |
| `locked_until` | `timestamptz` null | Sign-in for the address is refused while `now() < locked_until` |
| `updated_at` | `timestamptz` not null | Last change |

## Functions

- `sign_in_is_locked(key) → boolean`: true while `locked_until > now()`.
- `sign_in_record_failure(key) → void`: atomically — no row: failures 1. Locked: unchanged. Lock expired: failures 1, lock cleared. Otherwise failures + 1; at 5, `locked_until = now() + 15 minutes`.
- `sign_in_clear(key) → void`: deletes the row.
- `sign_in_record_failure` also deletes up to 100 rows untouched for 30 days (indexed on `updated_at`), so the table does not keep every address that ever failed once.

## Transitions

```text
(no row) --fail--> 1 --fail--> … 4 --fail--> 5, locked 15 min
locked --any attempt--> refused, unchanged
locked --15 min--> next fail → 1 (lock cleared)
any --success or completed reset--> (no row)
```

## Privileges

`vaultory_auth` gets `SELECT, INSERT, UPDATE, DELETE` on the table and `EXECUTE` on the three functions — nothing else changes for it.
