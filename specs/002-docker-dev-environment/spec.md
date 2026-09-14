# Feature Specification: Docker Development Environment

**Feature Branch**: `002-docker-dev-environment`

**Created**: 2026-09-14

**Status**: Draft

**Input**: User description: "I want it to be docker friendly"

## Clarifications

### Session 2026-09-14

- Q: What does "Docker-friendly" cover — just the database, or the whole stack? → A: The whole
  stack. Today `docker-compose.yml` provides PostgreSQL only, and both applications must be run
  from the host with Go, Node, and `golang-migrate` installed locally. The goal is that a clone of
  this repository with Docker installed and nothing else can run Vaultory and its full test suite.
- Q: Is this a product feature? → A: No — it is developer tooling. It changes nothing a collector
  sees, adds no requirement to feature 001, and touches no application source. It is specified
  separately because the constitution asks that significant work begin from a written
  specification, and because it has acceptance criteria worth stating.

## User Scenarios & Testing *(mandatory)*

The actor throughout is a **developer**, not a collector. Vaultory's collector-facing behaviour is
unchanged by this feature.

### User Story 1 - Run the whole stack from a clean machine (Priority: P1)

A developer clones Vaultory onto a machine that has Docker and nothing else — no Go toolchain, no
Node, no PostgreSQL, no `golang-migrate`. They run a single command and, after it finishes, open a
browser to a working Vaultory with a database already migrated and a collector they can sign in as.

**Why this priority**: This is the whole point. Everything else in this feature is in service of a
developer being able to run the project without first installing four things and reconciling a
connection string.

**Independent Test**: On a machine with only Docker, clone the repository, run the documented
command, and confirm the collection view loads and a collectible can be added. Delivers a working
environment on its own.

**Acceptance Scenarios**:

1. **Given** a machine with Docker installed and no Go, Node, PostgreSQL, or migration tool,
   **When** the developer runs the documented start command from a fresh clone, **Then** the
   frontend, the backend, and the database all start, migrations are applied, and the collection
   view loads in a browser.
2. **Given** the stack is running, **When** the developer signs in as the development collector and
   adds a collectible with a photograph, **Then** it is saved and appears in the gallery with its
   image, exactly as it does when run from the host.
3. **Given** the stack has been started once, **When** the developer stops it and starts it again,
   **Then** their collectibles and uploaded images are still there.
4. **Given** the stack is running, **When** the developer edits a source file, **Then** they can
   see the change without rebuilding an image from scratch.

---

### User Story 2 - Run the full test suite without installing toolchains (Priority: P2)

A developer runs Vaultory's complete test suite — including the integration, contract, and
end-to-end tests that need a real PostgreSQL — using only Docker. This is what currently blocks
tasks T088 and T089 of feature 001 from ever being closed on a machine without a database.

**Why this priority**: Roughly forty per cent of Vaultory's tests have never run anywhere, because
running them requires PostgreSQL, a migration tool, and both toolchains. This makes those tests
reachable by anyone.

**Independent Test**: On a machine with only Docker, run the documented test command and confirm
the integration, contract, and end-to-end suites all execute and report results.

**Acceptance Scenarios**:

1. **Given** a machine with only Docker, **When** the developer runs the documented test command,
   **Then** the backend unit, integration, and contract suites all run against a real PostgreSQL
   and report pass or fail.
2. **Given** the same machine, **When** the developer runs the documented end-to-end command,
   **Then** the browser suite runs against a running stack and reports results.
3. **Given** a test run has finished, **When** the developer inspects their development database,
   **Then** it is unaffected — tests must not run against, or destroy, development data.

---

### User Story 3 - Build a production-shaped image (Priority: P3)

A developer produces a runnable image of each application that contains no build toolchain, no
source, and no development-only behaviour.

**Why this priority**: Valuable for deployment and for catching "works in development only"
problems early, but nobody is blocked on it today.

**Independent Test**: Build each image, run it against a database, and confirm it serves correctly
and refuses to start with development identity enabled.

**Acceptance Scenarios**:

1. **Given** the production image definitions, **When** a developer builds them, **Then** each
   resulting image contains a compiled artefact and its runtime dependencies only — no compiler,
   no package manager cache, and no test files.
2. **Given** a production backend image, **When** it is started with the development identity
   setting enabled, **Then** it refuses to start and says why.

---

### Edge Cases

- A developer already has something bound to the database, backend, or frontend port — the failure
  must name the conflicting port rather than failing obscurely.
- A developer runs the start command twice — the second run must not duplicate data, re-seed over
  existing collectors, or fail on an already-applied migration.
- Migrations have not finished when the backend starts — the backend must not serve requests
  against an unmigrated database.
- The database container restarts — the backend must recover rather than stay permanently broken.
- A developer runs the test command while the development stack is running — neither must interfere
  with the other.
- Uploaded images must survive a container being rebuilt, or a developer loses their test data on
  every change.
- A developer on Apple Silicon and one on an x86 machine must both be able to build and run.
- A developer stops the stack and removes it entirely — there must be a documented way to discard
  all data deliberately, distinct from stopping.

## Requirements *(mandatory)*

### Functional Requirements

**Running the stack**

- **FR-001**: The repository MUST provide a single documented command that starts the database,
  the backend, and the frontend together.
- **FR-002**: Starting the stack MUST apply all outstanding database migrations before the backend
  begins serving requests.
- **FR-003**: The stack MUST require no host-installed Go, Node, PostgreSQL, or migration tool.
- **FR-004**: The stack MUST seed the development collectors, so a developer can sign in
  immediately and so cross-collector behaviour is exercisable.
- **FR-005**: Starting the stack a second time MUST be safe: no duplicated data, no failure on
  already-applied migrations.
- **FR-006**: Collector data and uploaded images MUST persist across a stop and start of the stack.
- **FR-007**: The stack MUST provide a documented way to discard all data deliberately, separate
  from stopping it.
- **FR-008**: A port conflict MUST fail with a message naming the port.
- **FR-009**: Every port the stack binds MUST be overridable without editing a tracked file.

**Developing against it**

- **FR-010**: A source change MUST become visible without a full image rebuild.
- **FR-011**: Dependency caches MUST persist between runs, so a routine start does not re-download
  the dependency tree.
- **FR-012**: Application logs MUST be visible to the developer while the stack runs.

**Testing**

- **FR-013**: The repository MUST provide a documented command that runs the backend unit,
  integration, and contract suites against a real PostgreSQL, using only Docker.
- **FR-014**: The repository MUST provide a documented command that runs the end-to-end browser
  suite against a running stack, using only Docker.
- **FR-015**: Test runs MUST use a database separate from the development database, and MUST NOT
  alter development data.
- **FR-016**: A test command MUST exit non-zero when any test fails, so it is usable in automation.

**Production images**

- **FR-017**: Each application MUST have an image definition that produces a runtime image
  containing no compiler, no package manager cache, no source tree, and no test files.
- **FR-018**: Production images MUST NOT run as root.
- **FR-019**: A production backend image MUST refuse to start when development identity is
  enabled, and MUST say why.
- **FR-020**: Images MUST build on both arm64 and x86-64.

**Secrets and configuration**

- **FR-021**: No credential may be committed except values that are plainly development-only and
  documented as such.
- **FR-022**: Every configuration value the applications read MUST be settable through the Docker
  setup without editing application source.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On a machine with only Docker installed, a developer goes from `git clone` to a
  working Vaultory in the browser with one command and no manual configuration.
- **SC-002**: That first run completes in under 10 minutes on a normal connection; subsequent
  starts complete in under 60 seconds.
- **SC-003**: 100% of Vaultory's test suites — unit, integration, contract, and end-to-end — are
  runnable with only Docker installed, closing the gap that leaves feature 001 at 95 of 97 tasks.
- **SC-004**: A source change is reflected in the running stack within 10 seconds, without an
  image rebuild.
- **SC-005**: Stopping and restarting the stack preserves 100% of collectibles and uploaded images.
- **SC-006**: Each production image is under 150 MB and contains no compiler or package manager.
- **SC-007**: The documented commands work unchanged on arm64 and x86-64.
- **SC-008**: A test run leaves the development database byte-identical to how it was before.

## Assumptions

- Docker Desktop or an equivalent providing `docker` and `docker compose` v2 is the only
  prerequisite. Podman compatibility is not a goal.
- The existing `docker-compose.yml`, which provides PostgreSQL alone, is superseded by this work.
- Development and production concerns are both in scope, but no deployment target, registry,
  orchestration platform, or CI pipeline is chosen here.
- Running from the host stays supported. This adds a way to run Vaultory; it does not remove one.
- No application source changes. If this feature cannot be delivered without editing `backend/` or
  `frontend/` application code, that is a finding to raise rather than a change to make quietly —
  the one anticipated exception is the frontend build configuration, which may need an output mode
  suited to containers.

**Scope boundaries** — explicitly excluded:

- Deploying anywhere, or choosing a host
- A CI pipeline or workflow files
- An image registry, tagging scheme, or release process
- Kubernetes or any orchestrator beyond Compose
- Production secret management
- Changes to Vaultory's collector-facing behaviour
