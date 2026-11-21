# Research: Audit Log

## 1. One table, two writers
- **Decision**: `audit_events(id, occurred_at, action, outcome, account_id, subject_id)`, with `action` and `outcome` constrained to known values.
- **Rationale**: one place to read; the account (Better Auth user id) is the common key across both services. For collection events the Go service has the collector, so the insert resolves `collectors.user_id` itself.
- **Alternatives**: log lines only (what FR-028 of feature 004 does — not durable, not append-only); a table per service (two places to read for one investigation).

## 2. Append-only, enforced by the database
- **Decision**: `BEFORE UPDATE` raises; `BEFORE DELETE` raises unless the row is over a year old; `BEFORE TRUNCATE` raises. The frontend's role has `INSERT` only.
- **Rationale**: FR-006. The Go service connects as the table's owner, so privileges alone cannot stop it; triggers apply to the owner too.
- **Alternatives**: revoking privileges from the owner (not possible); trusting the code (not evidence).

## 3. Collection events in the change's transaction
- **Decision**: `Add`, `Edit` and `Delete` insert their entry before committing.
- **Rationale**: FR-004 — a change and its record together or not at all; a refused change has no transaction to commit, so nothing is recorded (US2/AC2).

## 4. Account events from Better Auth
- **Decision**: registration from `databaseHooks.user.create.after`; sign-in success and failure from `hooks.after` on `/sign-in/email`; a locked refusal from `hooks.before`; sign-out from `databaseHooks.session.delete.after` when the path is `/sign-out` (so sessions ended by a password reset are not sign-outs); email verified from `databaseHooks.user.update.after` on `/verify-email`; reset completed from `onPasswordReset`.
- **Rationale**: each is the one place that knows the account and the outcome. Failures carry feature 013's address key, never the address.
- **Write failures**: caught and logged as a structured line naming the action, never thrown (FR-005).

## 5. Retention without a schedule
- **Decision**: `audit_purge_expired(500)` at the Go service's start-up and after each collectible deletion.
- **Rationale**: FR-007 and the repository's no-ticker convention (feature 006 Decision 5). An index on `occurred_at` keeps the check cheap however large the table is — the lesson of #72.

## 6. Migration number
- **Decision**: `000014`, after #121's `000013`; this branch is stacked on #121.
