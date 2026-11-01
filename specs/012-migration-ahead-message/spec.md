# Feature Specification: Say What Is Wrong When the Database Is Ahead

**Feature Branch**: `62-migration-ahead-message`

**Created**: 2026-10-28

**Status**: Draft

**Input**: Issue #62. Review a branch that adds a migration, run the stack, switch back, and
`make up` fails with `service "migrate" didn't complete successfully: exit 1`. The reason is only in
`docker compose logs migrate`, and even there it is oblique: `no migration found for version 12`.
Nothing is broken and no data is lost, but nothing printed says so. Each of #57, #58 and #59 adds a
migration, so every reviewer will meet this.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Told plainly what happened and what to do (Priority: P1)

A developer reviews a branch with a migration, runs the stack, switches back to `main` and runs
`make up`. Instead of an opaque failure, they are told in plain words: the database is at migration
12, this checkout only knows up to 10, that happened because another branch's migrations were
applied, their data is intact, and the one command that resolves it.

**Why this priority**: The whole of the issue.

**Independent Test**: Migrate a development database to a version higher than the newest migration in
the checkout, then start the migrations step. Its output names both versions, the cause, that data
is intact, and the resolving command, without needing `docker compose logs`.

**Acceptance Scenarios**:

1. **Given** a database ahead of the checkout, **When** the developer runs `make up`, **Then** the
   message is printed where they ran it, and nothing is applied, rolled back or changed.
2. **Given** that message, **When** the developer follows it, **Then** the stack starts.

---

### User Story 2 - The resolving command exists (Priority: P1)

The fix is a make target, not a line of Docker to assemble: from the branch whose migrations are
applied, move the database back to the version the other checkout knows.

**Why this priority**: The issue asks for the command to be offered as a make target.

**Independent Test**: From a checkout containing migrations up to 12, with the database at 12, run the
target to version 10; the database is at 10, and `make up` on a checkout ending at 10 then succeeds.

**Acceptance Scenarios**:

1. **Given** a checkout holding the down migrations, **When** the developer runs the target with a
   version, **Then** the database moves to exactly that version, through the checkout's own down
   migrations.
2. **Given** the target run without a version, **When** it starts, **Then** it refuses and says how
   to use it, changing nothing.

---

### Edge Cases

- **A database at or behind the checkout**: behaves exactly as today — pending migrations applied.
- **An empty database** (first run): behaves exactly as today.
- **A dirty database** (a migration failed half-way): reported as dirty with its version, not as
  "ahead"; its fix is different, and pretending otherwise would send the developer the wrong way.
- **The test database**: lives in tmpfs and is created fresh each run, so it cannot be ahead; out of
  scope.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Before applying anything, the migrations step MUST compare the database's recorded
  version with the newest migration in the checkout.
- **FR-002**: When the database is ahead, the step MUST stop without changing anything and print: the
  database's version; the newest version in the checkout; that this happens when another branch's
  migrations have been applied; that the data is intact; and the command that resolves it.
- **FR-003**: That message MUST be visible in the output of `make up` itself, not only in the
  migrations container's log.
- **FR-004**: A make target MUST move the development database to a given version using the current
  checkout's migrations, and MUST refuse to run without a version.
- **FR-005**: Nothing MUST be reversed automatically. Which way to go is the developer's choice.
- **FR-006**: A dirty database MUST be reported as dirty, with its version, and not as ahead.
- **FR-007**: In every other case, the step MUST behave exactly as it does today.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer who meets this state learns the cause and the fix from `make up`'s own
  output, with no other command, in under a minute.
- **SC-002**: 0 changes are made to the database when it is ahead.
- **SC-003**: Following the printed instructions starts the stack in 100% of cases where the newer
  branch is still available locally.

## Clarifications

### Session 2026-10-28

- Q: What is "the one command that resolves it", given this checkout lacks the newer migrations' down
  files? → A: The down files live on the branch that added them, so the fix runs there:
  `make migrate-to VERSION=<checkout's newest>`, then switch back. The message says so in those
  steps, and also names the other ways out — go forward to that branch again, or `make reset`, which
  destroys data — so nobody is pushed into the destructive one.

## Assumptions

- golang-migrate 4.18.1, as pinned in `compose.yaml`; its `version` command prints the version on
  stderr and exits 0, or prints `error: no migration` and exits 1 on an empty database. Both were
  observed, not assumed.
- Migration files are named `NNNNNN_name.up.sql`, as all of them are.
- Spec directory 012: 009–011 are in use on open branches.
