# Implementation Plan: Say What Is Wrong When the Database Is Ahead

**Branch**: `62-migration-ahead-message` · **Spec**: [spec.md](spec.md) · **Issue**: #62
**Date**: 2026-10-28 · **Status**: planned

## Summary

The `migrate` one-shot runs a small POSIX shell script instead of calling `migrate up` directly. The
script reads the database's version, finds the newest migration in the checkout, and either runs
`up` as before or stops with a plain explanation. `make up` prints the migrations step's output when
that step fails. A new `make migrate-to VERSION=n` moves the database to a version through the same
script.

## Technical Context

| | |
|---|---|
| **Changed** | `compose.yaml` (the `migrate` service's entrypoint), `Makefile` (`up`, new `migrate-to`), `backend/scripts/migrate.sh` (new), README |
| **Application code** | None |
| **Runtime** | The `migrate/migrate:v4.18.1` image: Alpine, busybox `sh`, the `migrate` binary |
| **Testing** | The script runs on the host against a real PostgreSQL with the same migrate version, in every state: ahead, dirty, empty, behind, current, and the move itself. Compose and make cannot run here (no Docker) |

## Constitution Check

| Principle | Assessment |
|---|---|
| **I–III** | Not affected: no product, application or contract change |
| **IV. Data Integrity** | The script never changes a database that is ahead or dirty (SC-002). The only command that moves a database backward is the explicit `migrate-to`, run by the developer |
| **V. Simplicity** | One shell script in the image already used, no new image or tool. Make stays thin; the logic lives in the script, where it can be read |

**Gate**: passes.

## Decisions

1. **A wrapper script, not a separate check service.** Dependents already wait on `migrate`
   completing, so the check belongs inside that step; a separate service would be a second ordering
   to keep right.
2. **Read the version with `migrate version`, not SQL.** The image has no `psql`, and the CLI owns
   the meaning of its own table. It prints the version on stderr (exit 0), `N (dirty)` when dirty,
   and `error: no migration` (exit 1) on an empty database. All three were observed with 4.18.1.
3. **The checkout's newest version from file names** (`NNNNNN_*.up.sql`), compared as integers.
4. **`make up` shows the step's output on failure** by reading the `migrate` container's exit code
   and printing its log without prefixes. Only that step's log, and only when it failed, so an
   unrelated failure is not buried under it.
5. **Exit codes**: 0 applied or current; 3 database ahead; 4 dirty; anything else is migrate's own.

## Project Structure

```text
specs/012-migration-ahead-message/ — spec, plan, quickstart, tasks, checklists/
backend/scripts/migrate.sh   new
compose.yaml                 migrate: entrypoint, script mount, URL in the environment
Makefile                     up shows the step's output on failure; migrate-to
README.md                    one paragraph under running locally
```

No data model, no contracts.
