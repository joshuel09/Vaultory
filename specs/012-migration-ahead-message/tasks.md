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

## Phase 3: Convergence

Issue #77, found by review of #76. Scope: FR-008's guard and its two gaps.

- [X] T011 `backend/scripts/migrate.sh`: cut the authority at the first `/`, `?` or `#`, not only `/`, so a query string can neither hide a password from the user-part check nor make an `@` in it look like one, per FR-008 (partial)
- [X] T012 `backend/scripts/migrate.sh`: refuse, without printing the address, a query parameter whose key decodes to `password` — percent-encoded letters and any case included — and keep accepting every other key, such as `sslmode`, per FR-008 (missing)
- [X] T013 Verify on the host against PostgreSQL 17 with password authentication and migrate 4.18.1: that lib/pq does read `?password=` (so the gap is real); that the guard refuses it plain, percent-encoded and in upper case with the password printed 0 times; that `?sslmode=disable` alone, `password` as a value or as part of a longer key, and an `@` inside the query are still accepted; and every case from T010 again, per FR-008, SC-004 (missing)


**T013 as run** (2026-11-01, PostgreSQL 17.6 with `scram-sha-256`, migrate 4.18.1, a dummy password). The gap is real: with no guard, migrate connects with `?password=` and, given a wrong one, prints it once. With the guard, `password`, `p%61ssword`, `%70%41%73%73%77%4F%72%64`, `PASSWORD`, a `;` separator and a bare `password` key are all refused with exit 2, as are a user-part password with and without a path; the password appears 0 times in each. Accepted: `sslmode` alone, `password` as a value, `passwordx` as a key, an `@` and `:` inside the query (with and without a path — before T011, the no-path form was wrongly refused), and a `#password=` fragment. T010's cases again: wrong `PGPASSWORD` prints no password; `goto 9` and `up` work.

## Phase 4: Convergence

Issue #70. Scope: FR-003 as amended — which migrate run `make up` reports.

- [X] T014 `Makefile`: compare the migration container StartedAt before and after startup, and read logs since the new nanosecond timestamp only when that run failed (FR-003; #70, #81).
- [X] T015 Verify `up`'s failure path with a stand-in for `docker compose` that answers `up`, `ps` and `logs --since` as each case would: an unrelated failure with a leftover failed `migrate` (nothing printed), migrate failing in this run (printed), migrate succeeding in this run under an unrelated failure (nothing printed), and success (no log); the exit status is always `up`'s own, per FR-003 (missing)


**T015 as run** (2026-11-01, GNU Make 3.81, a shell stand-in for `docker compose` answering `up`, `ps` and `logs --since`): an unrelated failure with a leftover `migrate` that exited 3 prints nothing (the recipe before this change printed that stale log); migrate failing in this run prints its log; migrate succeeding in this run under an unrelated failure prints nothing; no migrate container prints nothing; success prints the banner; `make` fails in every failing case, as before. `--since` received `2026-11-01T12:13:18Z`. Not run here: real Docker, so whether the daemon's log timestamps and the host clock agree to the second is not measured — Docker Desktop keeps its VM's clock in sync with the host.

**Review verification (2026-11-12):** Six mock-Docker startup cases passed: stale failure suppressed, current failure printed, successful migration suppressed under unrelated failure, absent container suppressed, success banner, and newly created failure printed. Same-second nanosecond timestamps distinguish runs. Real Docker startup was not repeated in this review.
