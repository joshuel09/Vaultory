# Tasks: Say What Is Wrong When the Database Is Ahead

- [X] T001 *As built: also passes `force N` through, because T005 showed the dirty message's first draft recommended `goto`, which golang-migrate refuses on a dirty database.* `backend/scripts/migrate.sh`: read the version, find the checkout's newest; up when empty, behind or current; stop with exit 3 and the explanation when ahead; exit 4 with its own explanation when dirty; `goto N` passes through; any other argument refused; never prints the database URL, which carries the password (FR-001, FR-002, FR-005–FR-007; analyze A1)
- [X] T002 `compose.yaml`: the `migrate` service runs the script, with the script mounted read-only and the database URL in the environment (FR-001)
- [X] T003 `Makefile`: `up` prints the `migrate` step's output when that step exited non-zero; `migrate-to` requires `VERSION` and runs the script's `goto` (FR-003, FR-004)
- [X] T004 README: what this message means and what to do, under running locally
- [X] T005 Verify the script on the host against PostgreSQL 17 with migrate 4.18.1 in every state — ahead (nothing changed), dirty, empty, behind, current, `goto` both ways, no version — and record the output (SC-001, SC-002)
- [X] T006 Note in the PR what could not run here: `make`, Compose, and the image's busybox `sh`

## Phase 2: Convergence

Issue #69. Scope: FR-008 and SC-004 — the password in the migrations step's output.

- [X] T007 `compose.yaml`: take the password out of `MIGRATE_DATABASE_URL` and pass it as `PGPASSWORD`, which lib/pq reads, so golang-migrate's errors have no password to quote — including the `exec`ed `up`, whose output the script never sees, per FR-008 (missing)
- [X] T008 `backend/scripts/migrate.sh`: refuse, with exit 2 and without printing the address, an `MIGRATE_DATABASE_URL` whose user part carries a password; correct the header comment, which says the URL carries the password, per FR-008 (missing)
- [X] T009 `compose.prod.yaml`: the production `migrate` passes the password inside `-database` on its command line, the same leak in `docker compose logs migrate`; move it to `PGPASSWORD` as `auth-role` already does, per FR-008, SC-004 (missing)
- [X] T010 Verify on the host against PostgreSQL 17 with password authentication and migrate 4.18.1: right password from `PGPASSWORD` migrates; a wrong one fails with output containing the password 0 times, from both `version` and the `exec`ed `up`; an address with a password is refused unprinted, per SC-004 (missing)


**T010 as run** (2026-10-31, PostgreSQL 17.6 with `scram-sha-256`, migrate 4.18.1, a dummy password): wrong password with nothing applied, and wrong password at migration 10 — both fail authentication, and the quoted address is `postgres://vaultory@127.0.0.1:55433/…`; right password from `PGPASSWORD` applies 1–10, then `no change`; `goto 9` and back `up`; an address with a password refused with exit 2. The password appears 0 times in every case. The same `migrate version` with the password in the address, as before this change, printed it once. The production form (`-database` without a password, `PGPASSWORD` set) behaves the same both ways. Not run here: Compose, and the migrate image's busybox `sh` — the guard uses only POSIX parameter expansion.
